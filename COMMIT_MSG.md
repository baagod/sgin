fix(handler): 修复参数绑定中校验错误未及时返回的问题

在 `bindV3` 中，当 `Validator.ValidateStruct` 返回校验错误时，虽然进行了翻译处理，但未执行 `return`，导致程序继续执行并可能返回不正确的成功结果。
