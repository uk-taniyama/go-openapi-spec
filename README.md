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

## やりたいこと

- responses/./dscriptionの書き出し.....って何をどこから出せばいいんだ？？？
- code生成
- descriptionの書き出し
- メンバー名を小文字に
