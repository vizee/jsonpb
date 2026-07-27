# jsonpb

元数据驱动的 JSON ↔ Protobuf 转码库。通过一份用 Go 结构体描述的消息元数据（`Message` / `Field`），在 JSON 文本与 protobuf wire 字节之间直接互转，无需生成代码、无需 `reflect`、无需 `protoc` 生成的类型。

适用于网关/代理场景下把上游 JSON 透转为下游 protobuf，或反向把 protobuf 编码为 JSON。

## 特性

- **零代码生成**：用 `Message`/`Field` 描述结构，运行时即可转码。
- **双向转码**：`TranscodeToJson`（proto->json）、`TranscodeToProto`（json->proto）。
- **低开销**：流式解析 + 直接写入，避免中间反射与结构体装配；`JsonBuilder` 支持复用已有缓冲。
- **proto3 语义**：默认值省略、`last-one-wins`、repeated 字段跨非连续出现拼接、同时接受 packed/unpacked 标量。
- **完备的标量/容器类型**：12 种数值类型、bool、string、bytes、map、嵌套 message、repeated。

## 安装

```bash
go get github.com/vizee/jsonpb
```

要求 Go ≥ 1.23，依赖 `google.golang.org/protobuf`（仅用其 `protowire` 编解码原语）。

## 包结构

| 包 | 说明 |
|---|---|
| `github.com/vizee/jsonpb` | 顶层转码入口与消息元数据定义 |
| `github.com/vizee/jsonpb/proto` | protobuf wire 格式的 `Encoder` / `Decoder` |
| `github.com/vizee/jsonpb/jsonlit` | 流式 JSON 词法迭代器 `Iter` 与字符串转义工具 |

## 快速开始

### 定义元数据

```go
import "github.com/vizee/jsonpb"

var SimpleMsg = jsonpb.NewMessage("Simple", []jsonpb.Field{
    {Name: "name", Tag: 1, Kind: jsonpb.StringKind, Omit: jsonpb.OmitEmpty},
    {Name: "age",  Tag: 2, Kind: jsonpb.Int32Kind,   Omit: jsonpb.OmitEmpty},
    {Name: "male", Tag: 3, Kind: jsonpb.BoolKind,    Omit: jsonpb.OmitAlways},
}, true, true)
```

`NewMessage` 的最后两个参数控制是否构建 tag 索引与 name 索引（建议都传 `true`，可显著加速按 tag/按 name 查找）。

### JSON -> Protobuf

```go
import (
    "github.com/vizee/jsonpb"
    "github.com/vizee/jsonpb/jsonlit"
    "github.com/vizee/jsonpb/proto"
)

var enc proto.Encoder
it := jsonlit.NewIter([]byte(`{"name":"bob","age":23,"male":true}`))
if err := jsonpb.TranscodeToProto(&enc, it, SimpleMsg); err != nil {
    // ...
}
pb := enc.Bytes() // => 0a03626f621017
```

### Protobuf -> JSON

```go
var j jsonpb.JsonBuilder
if err := jsonpb.TranscodeToJson(&j, proto.NewDecoder(pb), SimpleMsg); err != nil {
    // ...
}
jsonStr := j.String() // => {"name":"bob","age":23}
```

### 复用缓冲

```go
// JsonBuilder 可基于已有 []byte 构建，避免重复分配
b := jsonpb.UnsafeJsonBuilder(make([]byte, 0, 256))
if err := jsonpb.TranscodeToJson(b, proto.NewDecoder(pb), SimpleMsg); err != nil {
    // ...
}
out := b.IntoBytes() // 取出并清空内部缓冲
```

## 元数据参考

### `Field`

| 字段 | 类型 | 说明 |
|---|---|---|
| `Name` | `string` | JSON 字段名 |
| `Tag` | `uint32` | protobuf 字段号（tag） |
| `Kind` | `Kind` | 字段类型 |
| `Repeated` | `bool` | 是否为重复字段 |
| `Ref` | `*Message` | `MapKind` 指向 map entry（含 tag=1 的 key 与 tag=2 的 value）；`MessageKind` 指向子消息 |
| `Omit` | `OmitRule` | 省略规则（见下） |

### `Kind`

`DoubleKind` `FloatKind` `Int32Kind` `Int64Kind` `Uint32Kind` `Uint64Kind` `Sint32Kind` `Sint64Kind` `Fixed32Kind` `Fixed64Kind` `Sfixed32Kind` `Sfixed64Kind` `BoolKind` `StringKind` `BytesKind` `MapKind` `MessageKind`。

`IsNumericKind(k)` 判断是否为数值类型（`DoubleKind..Sfixed64Kind`）。

### `OmitRule`

| 规则 | proto->json | json->proto |
|---|---|---|
| `OmitProtoEmpty`（默认） | 字段缺失时输出默认值（`0`/`false`/`""`/`{}`/`[]`） | - |
| `OmitEmpty` | 字段缺失时整段省略 | 空/零值字段不写入 wire |
| `OmitAlways` | 永不输出 | 永不写入 |

### Map

map 字段用 `MapKind` + `Ref` 指向一个 entry 消息，entry 固定含 tag=1（key）与 tag=2（value）：

```go
entry := jsonpb.NewMessage("", []jsonpb.Field{
    {Tag: 1, Kind: jsonpb.StringKind}, // key
    {Tag: 2, Kind: jsonpb.Int32Kind},  // value
}, true, true)

msg := jsonpb.NewMessage("M", []jsonpb.Field{
    {Name: "m", Tag: 1, Kind: jsonpb.MapKind, Ref: entry},
}, true, true)
```

map 的 key 可为字符串或整数类型；JSON 中数值 key 以字符串形式书写（如 `{"0":5}`）。

## 行为与语义

- **默认值省略**：json->proto 方向，标量的零值、空字符串/bytes、`false`、空消息不写入 wire（proto3 默认值不序列化）。`bytes`/`string` 以 base64（标准 padding）编码。
- **输出顺序**：proto->json 按字段定义顺序输出（含未出现字段的默认值，受 `OmitRule` 控制）。
- **repeated 字段**：proto->json 同时接受 packed 与 unpacked 两种编码并拼接所有出现；json->proto 数值 repeated 一律输出为 packed。
- **非重复字段重复出现**：proto->json 取最后一次出现（last-one-wins）。
- **特殊浮点值**：proto->json 输出 `NaN` / `Infinity` / `-Infinity`（遵循 protobuf JSON 规范）。
- **JSON 词法**：json->proto 的词法分析为性能做了取舍，不完全按 JSON 标准做语法校验（如允许部分分隔符缺省），但数值/字符串仍按类型严格解析。
- **map entry**：key 始终写出（即使为空串或 0），保证默认 key + 默认 value 的条目不丢失；value 缺失时取默认值。

## 限制

- 非重复 **message** 字段重复出现时为 last-one-wins（末条整体覆盖），未实现 proto3 字段级 merge。
- map 同名 key 重复时按出现顺序拼接（会产生重复 JSON 键），未做 last-one-wins 去重。
- 不支持 protobuf group（proto2）。
- JSON 解析非严格标准（见上）。

## 许可证

见 [LICENSE](LICENSE)。
