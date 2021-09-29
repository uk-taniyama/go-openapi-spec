# go-openapi-spec
openapi spec by go

goでOpenAPIのスペック(サブセット)を書いてOpenAPI spec jsonを出力する。

とりあえず、なんかはできた。
ところどころ動かんけどな！

## できたこと

- infoの埋め込み→const OpenAPISpecにspecを書く！
- validationの書き出し→StructTagに生json5で書く！
- responses/./dscriptionの書き出し.....って何をどこから出せばいいんだ？？？→コメントから取得する！
- yamlで出しているけど、トップの出力だけでも何とかしたいものだ......→yaml.MapSliceで順序指定！
- メンバー名を小文字に
- Secirityの書き出し→const Authにspecを書く。

## やりたいこと

- code生成
- descriptionの書き出し
- tagのサポート。interfaceをtagにしてみよう。