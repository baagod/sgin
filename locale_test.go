package sgin

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"golang.org/x/text/language"
)

func TestSetupLocaleComponents(t *testing.T) {
	// 创建校验器实例
	validate := validator.New()

	// 测试中文和英文
	tags := []language.Tag{language.Chinese, language.English}

	defaultLang, matcher, translator := localeComponents(validate, tags...)

	// 检查 matcher 是否创建
	if matcher == nil {
		t.Error("expected non-nil language matcher")
	}

	// 检查 translator 是否创建
	if translator == nil {
		t.Error("expected non-nil translator")
	}

	// 检查 defaultLang 应该是中文（第一个）
	defaultBase, _ := defaultLang.Base()
	chineseBase, _ := language.Chinese.Base()
	if defaultBase != chineseBase {
		t.Errorf("expected default language Chinese, got %v (base %v)", defaultLang, defaultBase)
	}

	// 测试语言匹配功能
	if matcher != nil {
		preferred := []language.Tag{language.MustParse("zh-CN")}
		matched, _, _ := matcher.Match(preferred...)
		matchedBase, _ := matched.Base()
		chineseBase, _ := language.Chinese.Base()
		if matchedBase != chineseBase {
			t.Errorf("expected matched language Chinese for zh-CN, got %v (base %v)", matched, matchedBase)
		}

		// 测试英文匹配
		preferredEn := []language.Tag{language.MustParse("en-US")}
		matchedEn, _, _ := matcher.Match(preferredEn...)
		matchedEnBase, _ := matchedEn.Base()
		englishBase, _ := language.English.Base()
		if matchedEnBase != englishBase {
			t.Errorf("expected matched language English for en-US, got %v (base %v)", matchedEn, matchedEnBase)
		}
	}

	// 测试不支持的语言（如日语，未配置）
	tagsWithUnsupported := []language.Tag{language.Chinese, language.Japanese}
	defaultLang2, matcher2, translator2 := localeComponents(validate, tagsWithUnsupported...)
	// 应该只支持中文，跳过日语
	if defaultLang2 != language.Chinese {
		t.Errorf("expected default language Chinese when Japanese unsupported, got %v", defaultLang2)
	}
	if matcher2 == nil {
		t.Error("expected non-nil matcher even with unsupported language")
	}
	if translator2 == nil {
		t.Error("expected non-nil translator even with unsupported language")
	}
}

func TestSetupLocaleComponentsWithEmptyTags(t *testing.T) {
	validate := validator.New()
	defaultLang, matcher, translator := localeComponents(validate)
	// 空 tags 应返回零值
	if defaultLang != language.Und {
		t.Errorf("expected Und default language for empty tags, got %v", defaultLang)
	}
	if matcher != nil {
		t.Error("expected nil matcher for empty tags")
	}
	if translator != nil {
		t.Error("expected nil translator for empty tags")
	}
}

func TestSetupLocaleComponentsWithNilValidator(t *testing.T) {
	// 测试 validate 为 nil 的情况
	tags := []language.Tag{language.Chinese, language.English}
	defaultLang, matcher, translator := localeComponents(nil, tags...)

	// validate 为 nil 时应直接返回零值
	if defaultLang != language.Und {
		t.Errorf("expected Und default language for nil validator, got %v", defaultLang)
	}
	if matcher != nil {
		t.Error("expected nil matcher for nil validator")
	}
	if translator != nil {
		t.Error("expected nil translator for nil validator")
	}
}
