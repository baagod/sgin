package sgin

import (
	"github.com/go-playground/locales/de"
	"github.com/go-playground/locales/es"
	"github.com/go-playground/locales/fr"
	"github.com/go-playground/locales/ja"
	"github.com/go-playground/locales/ko"
	"github.com/go-playground/locales/ru"
	"github.com/go-playground/locales/zh_Hans"
	tfr "github.com/go-playground/validator/v10/translations/fr" // 法语
	tja "github.com/go-playground/validator/v10/translations/ja" // 日语
	tko "github.com/go-playground/validator/v10/translations/ko" // 韩语
	tru "github.com/go-playground/validator/v10/translations/ru" // 俄语
	"golang.org/x/text/language"

	"github.com/go-playground/locales"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	tde "github.com/go-playground/validator/v10/translations/de" // 德语
	ten "github.com/go-playground/validator/v10/translations/en" // 英语
	tes "github.com/go-playground/validator/v10/translations/es" // 西班牙语
	tzh "github.com/go-playground/validator/v10/translations/zh" // 中文
)

// langMapping 存储语言标签到翻译器构造函数的映射
var langMapping = map[language.Tag]struct {
	translator func() locales.Translator
	register   func(*validator.Validate, ut.Translator) error
}{
	language.Chinese:           {zh.New, tzh.RegisterDefaultTranslations},      // 中文
	language.SimplifiedChinese: {zh_Hans.New, tzh.RegisterDefaultTranslations}, // 简体中文
	language.English:           {en.New, ten.RegisterDefaultTranslations},      // 英文
	language.Japanese:          {ja.New, tja.RegisterDefaultTranslations},      // 日语
	language.Korean:            {ko.New, tko.RegisterDefaultTranslations},      // 韩语
	language.French:            {fr.New, tfr.RegisterDefaultTranslations},      // 法语
	language.Russian:           {ru.New, tru.RegisterDefaultTranslations},      // 俄语
	language.German:            {de.New, tde.RegisterDefaultTranslations},      // 德语
	language.Spanish:           {es.New, tes.RegisterDefaultTranslations},      // 西班牙语
}

// SupportedLanguages 返回框架支持的所有语言标签
func SupportedLanguages() []language.Tag {
	tags := make([]language.Tag, 0, len(langMapping))
	for tag := range langMapping {
		tags = append(tags, tag)
	}
	return tags
}

// localeComponents 根据语言标签创建多语言核心组件
func localeComponents(validate *validator.Validate, tags ...language.Tag) (
	defaultLang language.Tag,
	matcher language.Matcher,
	translator *ut.UniversalTranslator,
) {
	if len(tags) == 0 || validate == nil {
		return
	}
	var supportedTags []language.Tag

	for _, tag := range tags {
		m, ok := langMapping[tag]
		if !ok {
			debugWarning("language %s is not supported, skipping", tag.String())
			continue
		}

		// 不需要再 language.Parse，在 map 中的值必定是合法的。
		lang, locale := tag.String(), m.translator()
		if supportedTags = append(supportedTags, tag); translator == nil {
			translator = ut.New(locale)
		}
		_ = translator.AddTranslator(locale, true)

		trans, _ := translator.GetTranslator(lang)
		if err := m.register(validate, trans); err != nil {
			debugWarning("failed to register [%s] translator: %v", lang, err)
		}
	}

	// 创建语言匹配器
	if len(supportedTags) > 0 {
		defaultLang = supportedTags[0]
		matcher = language.NewMatcher(supportedTags)
	}

	return defaultLang, matcher, translator
}
