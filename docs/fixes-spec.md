# jsonpb 缺陷修复规格说明

- 状态：已实现
- 日期：2026-07-28
- 范围：`jsonlit`、`ptoj.go`（proto→json）、`jtop.go`（json→proto）、`metadata.go`
- 验证：`go test ./...` 全部通过（187 项），`go vet`/`gofmt` 干净

---

## 1. 背景

`jsonpb` 是一个元数据驱动的 JSON↔Protobuf 转码库。经源码审查发现 11 处偏离 JSON/Protobuf 规范或会导致数据损失的问题。本文件描述每个问题的根因、修复设计与行为变更。

两类公开入口：
- `TranscodeToJson(j *JsonBuilder, p *proto.Decoder, msg *Message)`：proto → json
- `TranscodeToProto(p *proto.Encoder, j *JsonIter, msg *Message)`：json → proto

---

## 2. 问题与修复

### #1 [高] JSON 字符串以反斜杠结尾时分词失败
- **位置**：`jsonlit/iter.go:45` `nextString`
- **根因**：闭合引号判定为 `s[p] == '"' && s[p-1] != '\\'`。当闭合引号前是一个"被转义的反斜杠"（即值以单个 `\` 结尾，如 `"\\"`、`"a\\"`、`"C:\\dir\\"`），`s[p-1]` 恰为 `\`，闭合引号被误判为"被转义"而跳过。
- **影响**：返回 `Invalid`（当作未终止），或越过闭合引号吞掉相邻 token。任何以反斜杠结尾的字符串值（Windows 路径、正则等）都会破坏解析。
- **修复**：改为正向追踪转义状态——遇到 `\` 时跳过下一字符，再判断 `"`。
- **验证**：`TestRegress_NextStringBackslash`；端到端 `{"s":"a\\"}` 往返正确。

### #2 [高] 重复字段在非连续出现时丢数据
- **位置**：`ptoj.go` `transProtoMessage`（旧实现用 `emitted` 去重，且 packed 仅处理单个分片）
- **根因**：流式输出 + `emitted[fieldIdx]` 去重，使重复字段的后续出现被跳过；`transProtoRepeatedBytes` 只消费连续同 tag 段，`transProtoPackedArray` 只处理一个 packed 分片。
- **影响**：同一重复字段在 wire 流中被其它字段分隔（合法且常见，如流拼接、`proto.Merge`）时，除第一组外全部丢失。违反 protobuf"重复字段所有出现须拼接"语义。
- **修复**：`transProtoMessage` 改为两遍处理：
  1. 收集每个字段的所有出现 `[][]fieldScan`；
  2. 按字段定义顺序输出，重复字段拼接所有出现，map 拼接所有 entry。
- **验证**：`TestRegress_RepeatedStringSplit`、`TestRegress_PackedSplit`、`TestRegress_MapNonConsecutive`。

### #3 [中] 不支持非 packed 的 repeated 标量
- **位置**：`ptoj.go:78` `acceptFieldWire`（新增），`ptoj.go:268` `transProtoRepeated`
- **根因**：旧 `getFieldWireType` 对 repeated 一律返回 `BytesType`，并强校验 wire 类型，拒绝 unpacked 编码。proto3 规范要求解析器**同时接受** packed 与 unpacked。
- **修复**：`acceptFieldWire` 对 repeated 标量接受 `BytesType`（packed）或标量自身 wire 类型（unpacked）；`transProtoRepeated` 对每个出现按 wire 类型分别解码（packed 分片逐元素、unpacked 单元素），并拼接。
- **验证**：`TestRegress_UnpackedRepeated`、`TestRegress_MixedPackedUnpacked`。

### #4 [中] Map 中 key 与 value 均为默认值时整条 entry 被丢弃
- **位置**：`jtop.go:222` `transJsonToMap`
- **根因**：entry buffer 先写 key 再写 value，最后 `if buf.Len() != 0 { EmitBytes }`。而 `transJsonString(omitEmpty=true)`/`transJsonNumeric`(跳过 0) 会对默认 key/value 各自省略；二者皆默认时 buffer 为空，entry 不输出。
- **影响**：`{"":{"":0}}` 类条目丢失，json→proto→json 不可逆。
- **修复**：map 的 key 始终写出（字符串 key `omitEmpty=false`，数值 key `transJsonNumeric(...,false)`）。value 仍可省略（缺失 value 在解码时取默认值，entry 由 key 保留）。`zero_value` 既有用例（value 省略）行为不变。
- **验证**：`TestRegress_MapDefaultEntryKept`；端到端 `{"m":{"":0,"a":5}}` 往返正确。

### #5 [中] `EscapeString` 不转义控制字符，产出非法 JSON
- **位置**：`jsonlit/string.go:28` `EscapeString`
- **根因**：`escapeTable` 对 0x00–0x1F 中除 `\b\t\n\f\r` 外的控制字符均为 `rawMark`，原样拷贝。JSON 规范要求 0x00–0x1F 必须转义。
- **修复**：对 `c < 0x20` 的字符输出 `\u00XX`。
- **验证**：`TestRegress_EscapeControl`。

### #6 [中] `\u` 代理对未合并，补充平面字符被损坏
- **位置**：`jsonlit/string.go:58` `UnescapeString`
- **根因**：每个 `\uXXXX` 独立 `utf8.EncodeRune`，不识别高位+低位代理组合。补充平面字符（如 emoji）以 `\uD83D\uDE00` 输入时，两个代理各自被编码为 U+FFFD。
- **修复**：识别高位代理（D800–DBFF）后消费紧随的低位代理（DC00–DFFF），合并为补充平面码点。孤位代理仍编码为替换字符（不报错）。
- **验证**：`TestRegress_SurrogatePair`。

### #7 [中] 非重复字段重复出现时取第一个而非最后一个
- **位置**：`ptoj.go` `transProtoMessage`（两遍处理）
- **根因**：旧 `emitted` 去重保留首次出现。proto3 对非重复标量为 last-one-wins，对消息为字段级合并。
- **修复**：两遍处理后对非重复字段取 `occ[len(occ)-1]`（last-one-wins）。注：消息字段的字段级合并未实现，当前为"末条整体覆盖"，较"首条胜出"更接近规范。
- **验证**：`TestRegress_LastOneWins`。

### #8 [低] 数值类型 map key 缺省时输出 `""` 而非 `"0"`
- **位置**：`ptoj.go:121` `transProtoMapEntry`
- **根因**：entry 中 key 字段缺失（默认 0）时，对数值 key 也走 `j.AppendString('""')` 分支。
- **修复**：按 key 类型输出默认值——数值 key 为 `"0"`，字符串 key 为 `""`。
- **验证**：`Test_transProtoMapCase` 的 `default_numeric_key` 用例。

### #9 [低] Double/Float 的 NaN/±Inf 产出非法 JSON
- **位置**：`ptoj.go:216` `appendFloat`（新增）
- **根因**：`strconv.AppendFloat('f')` 对 NaN/Inf 输出 `NaN`/`+Inf`/`-Inf`，非法 JSON。
- **修复**：新增 `appendFloat`，按 protobuf JSON 规范输出 `NaN`/`Infinity`/`-Infinity`，其余仍用最短 `'f'` 表示。
- **验证**：`TestRegress_FloatSpecial`。

### #10 [低] `FieldIndexByTag` 在 tag 含 0 时索引路径误判（潜在）
- **位置**：`metadata.go:54` `BakeTagIndex`、`metadata.go:86` `FieldIndexByTag`
- **根因**：旧实现用 `len(tagIdx) == len(Fields)` 区分 sparse（二分）与 dense（直接索引）。若 dense 分配恰好 `len(Fields)` 个槽（需要存在 tag 0），dense 数组会被当成 sparse 走二分；对含 `-1` 空槽的非法元数据（重复 tag）会越界 panic。
- **修复**：`Message` 新增显式 `tagIdxSparse bool` 标志，在 `BakeTagIndex` 中设置，`FieldIndexByTag` 据此选择路径。dense 路径对空槽 `-1` 返回 -1，不再 panic。
- **验证**：`TestMessage_denseWithTag0`、`TestMessage_duplicateTagNoPanic`。

### #11 [低] `transJsonNumeric` 零值省略判定不一致
- **位置**：`jtop.go:279` `isNumericZero`（新增）、`jtop.go:295` `transJsonNumeric`
- **根因**：仅 `len(s)==1 && s[0]=='0'` 省略；`0.0`/`0e0`/`-0`/`00` 仍写出 0（非规范但可往返）。
- **修复**：新增 `isNumericZero`——浮点类型接受 `0.0`/`0e0`/`-0` 等形式（更符合 proto3 默认值省略），整数类型仍仅 `"0"`，以保持对 `"0.0"` 这类非法整数字面量的严格报错。
- **验证**：`TestRegress_NumericZeroForms`。

---

## 3. 文件变更

| 文件 | 变更 |
|---|---|
| `jsonlit/iter.go` | 重写 `nextString`（#1） |
| `jsonlit/string.go` | `EscapeString` 控制字符转义（#5）；`UnescapeString` 代理对合并（#6）；新增 `hexVal`/`hexDigits` |
| `ptoj.go` | `transProtoMessage` 两遍重写（#2/#7）；新增 `acceptFieldWire`（#3）、`transProtoRepeated`/`transProtoSingular`/`transProtoMapEntry`/`fieldScan`/`appendFloat`（#8/#9）；移除旧流式 `transProtoRepeatedBytes`/`transProtoPackedArray`/`transProtoMap` |
| `jtop.go` | `transJsonToMap` key 非省略（#4）；`transJsonNumeric` 增 `omitEmpty` 参数与 `isNumericZero`（#4/#11） |
| `metadata.go` | `Message` 增 `tagIdxSparse`；`BakeTagIndex`/`FieldIndexByTag` 改用显式标志（#10） |

测试：
- 新增 `jsonlit/regress_test.go`（#1/#5/#6）
- 新增 `ptoj_regress_test.go`（#2/#3/#7/#9）
- 新增 `jtop_regress_test.go`（#4/#11）
- `metadata_test.go` 追加（#10）
- `ptoj_test.go` 更新：旧流式内部函数单测改为对新原语 `transProtoRepeated`/`transProtoMapEntry` 的等价覆盖；map 增 `default_numeric_key`（#8）
- `jtop_test.go`：`transJsonNumericCase` 适配新签名

---

## 4. 行为与兼容性说明

1. **proto→json 输出顺序**：由"wire 出现顺序"改为"字段定义顺序"。对 protobuf JSON 更规范；现有测试用例的 wire 本就按字段顺序编码，无回归。
2. **map value 省略**：保持原行为（默认 value 不写出，解码取默认值），`zero_value` 既有用例不变。仅 key 改为始终写出。
3. **非重复消息字段重复出现**：现为 last-one-wins（末条整体覆盖），非字段级合并。这是已知取舍，较旧实现（首条胜出）更接近规范。
4. **重复 tag 元数据**：非法但不再 panic（#10）。
5. **API 变更**：`transJsonNumeric` 新增 `omitEmpty` 参数（包内函数，非公开 API）。公开类型 `Message` 新增未导出字段 `tagIdxSparse`，向后兼容。

---

## 5. 验证

- `go test ./...`：187 项通过，0 失败。
- `go vet ./...`：无告警。
- `gofmt -l`：无输出。
- 端到端往返（探针，已移除）：
  - `{"s":"a\\"}` → `0a02615c` → `{"s":"a\\"}`（#1）
  - `{"m":{"":0,"a":5}}` → proto → `{"m":{"":0,"a":5}}`（#4）
  - `{"f":["a","b"]}` → `0a01610a0162`（#2）
  - `{"s":"😀"}` → `0a04f09f9880`（#6）

---

## 6. 后续可改进项（未在本轮处理）

- 非重复消息字段重复出现时的字段级合并（proto3 merge 语义）。
- map entry 同 key 时的 last-one-wins（当前为顺序拼接，重复 key 会产生重复 JSON 键）。
- JSON 词法分析仍非标准（`nextNumber` 宽松），见 `jtop.go` 既有注释。
