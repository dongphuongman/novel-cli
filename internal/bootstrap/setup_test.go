package bootstrap

import (
	"errors"
	"testing"

	"github.com/dongphuongman/novel-cli/internal/errs"
)

// TestEveryPresetResolvesAProtocol 是本文件最重要的一条，也是新增预设时最容易踩的坑。
//
// ProviderType() 只在两种情况下成功：显式钉死 type，或名字恰好在 agentcore 的
// knownProviders 里（12 个硬编码名字）。加一个网关预设却忘了钉 type，配置在
// 向导里能填完、能保存，直到真正建模型那一刻才炸——而且错误信息指向配置文件，
// 不指向这张预设表。这条测试把它挪到 CI。
func TestEveryPresetResolvesAProtocol(t *testing.T) {
	for _, preset := range ProviderPresets() {
		t.Run(preset.Name, func(t *testing.T) {
			if preset.NeedType {
				// Custom Proxy 的协议由用户在向导里选，预设本身无从判定。
				return
			}
			pc := ProviderConfig{Type: preset.ProviderType}
			got, err := pc.ProviderType(preset.Name)
			if err != nil {
				t.Fatalf("预设 %q 解析不出协议：%v\n"+
					"    要么这个名字得在 agentcore knownProviders 里，"+
					"要么预设必须钉死 providerType。", preset.Name, err)
			}
			if got == "" {
				t.Fatalf("预设 %q 解析出空协议", preset.Name)
			}
			if errors.Is(err, errs.ErrConfig) {
				t.Fatalf("预设 %q 返回了配置错误", preset.Name)
			}
		})
	}
}

// TestPinnedProtocolPresetsCarryAnAddress：钉死协议意味着这不是官方托管接口，
// 而是自建/代理端点——litellm 没有它的默认地址，预设不给就等于让用户自己猜。
func TestPinnedProtocolPresetsCarryAnAddress(t *testing.T) {
	for _, preset := range ProviderPresets() {
		if preset.ProviderType == "" || preset.NeedType {
			continue
		}
		if preset.BaseURL == "" {
			t.Errorf("预设 %q 钉死了协议 %q 却没有 base_url——"+
				"这类端点没有库内默认地址，必须预填", preset.Name, preset.ProviderType)
		}
	}
}

// TestRoutedGatewayPresetsSeedAModel：动态路由网关的价值就在于用户不必挑模型
// （"auto"），所以它必须自带默认模型；否则向导那一步会停在空输入框上。
func TestRoutedGatewayPresetsSeedAModel(t *testing.T) {
	for _, preset := range ProviderPresets() {
		if preset.RoutedGateway && preset.DefaultModel == "" {
			t.Errorf("预设 %q 标记为动态路由网关，却没有默认模型", preset.Name)
		}
	}
}

// TestProviderPresetsCopiesEveryField 防止 ProviderPresets() 漏抄新加的字段——
// 漏抄的表现是 /config 静默拿到零值（协议变空 → 建模型时报错），不是编译错误。
func TestProviderPresetsCopiesEveryField(t *testing.T) {
	exported := ProviderPresets()
	if len(exported) != len(setupProviders) {
		t.Fatalf("导出 %d 个预设，内部有 %d 个", len(exported), len(setupProviders))
	}
	for i, internal := range setupProviders {
		got := exported[i]
		switch {
		case got.Name != internal.name:
			t.Errorf("[%d] Name = %q，期望 %q", i, got.Name, internal.name)
		case got.Label != internal.label:
			t.Errorf("[%d] Label = %q，期望 %q", i, got.Label, internal.label)
		case got.BaseURL != internal.baseURL:
			t.Errorf("[%d] BaseURL = %q，期望 %q", i, got.BaseURL, internal.baseURL)
		case got.NeedType != internal.needType:
			t.Errorf("[%d] NeedType = %v，期望 %v", i, got.NeedType, internal.needType)
		case got.APIKeyOptional != internal.apiKeyOptional:
			t.Errorf("[%d] APIKeyOptional = %v，期望 %v", i, got.APIKeyOptional, internal.apiKeyOptional)
		case got.ProviderType != internal.providerType:
			t.Errorf("[%d] ProviderType = %q，期望 %q", i, got.ProviderType, internal.providerType)
		case got.DefaultModel != internal.defaultModel:
			t.Errorf("[%d] DefaultModel = %q，期望 %q", i, got.DefaultModel, internal.defaultModel)
		case got.RoutedGateway != internal.routedGateway:
			t.Errorf("[%d] RoutedGateway = %v，期望 %v", i, got.RoutedGateway, internal.routedGateway)
		}
	}
}

// TestFreeLLMAPIPreset 钉死这次接入的具体形状。
func TestFreeLLMAPIPreset(t *testing.T) {
	var preset ProviderPreset
	for _, p := range ProviderPresets() {
		if p.Name == "freellmapi" {
			preset = p
		}
	}
	if preset.Name == "" {
		t.Fatal("找不到 freellmapi 预设")
	}
	if preset.ProviderType != "openai" {
		t.Errorf("协议 = %q，期望 openai", preset.ProviderType)
	}
	if preset.BaseURL != "http://localhost:3001/v1" {
		t.Errorf("base_url = %q", preset.BaseURL)
	}
	if preset.DefaultModel != "auto" {
		t.Errorf("默认模型 = %q，期望 auto", preset.DefaultModel)
	}
	if preset.NeedType {
		t.Error("协议已钉死，不该再问用户")
	}
	if preset.APIKeyOptional {
		t.Error("FreeLLMAPI 需要 unified key，向导不该允许留空")
	}
}

// TestPresetModelSeedsJSONSchemaOnlyForRoutedGateways：种子模型的 json_schema
// 是三态指针，普通 provider 必须保持 nil（= 沿用适配器能力），只有动态路由网关
// 才显式写 false。写反了会悄悄关掉正常 provider 的原生结构化输出。
func TestPresetModelSeedsJSONSchemaOnlyForRoutedGateways(t *testing.T) {
	for _, sp := range setupProviders {
		t.Run(sp.name, func(t *testing.T) {
			m := presetModel(sp, "x")
			if m.Name != "x" {
				t.Fatalf("Name = %q", m.Name)
			}
			switch {
			case sp.routedGateway:
				if m.JSONSchema == nil || *m.JSONSchema {
					t.Errorf("动态路由网关应种下 json_schema:false，得到 %v", m.JSONSchema)
				}
			default:
				if m.JSONSchema != nil {
					t.Errorf("普通 provider 的 json_schema 应保持未设置，得到 %v", *m.JSONSchema)
				}
			}
		})
	}
}

// TestUnpinnedGatewayNameIsRejected 证明上面那条守卫不是摆设：
// 把 freellmapi 的 type 拿掉，ProviderType() 必须报错。
// 没有这条，TestEveryPresetResolvesAProtocol 可能因为"恰好都能过"而形同虚设。
func TestUnpinnedGatewayNameIsRejected(t *testing.T) {
	pc := ProviderConfig{} // 不钉 type
	if _, err := pc.ProviderType("freellmapi"); err == nil {
		t.Fatal("freellmapi 不在 agentcore knownProviders 里，不钉 type 就该报错——" +
			"如果这里不报错，说明上游把它加进注册表了，预设可以简化")
	}
}

// TestPresetHintsShowTheDefaultValue：向导里"有默认值的输入框"必须把默认值本身
// 当提示语，而不是泛例。
//
// 这条来自实测：预填生效了（回车确实得到 auto），但屏幕上显示的是
// "例如：gpt-4o / claude-sonnet-4 / …"，用户没有任何线索知道直接回车会得到什么。
// 功能对、界面不对，是最容易漏测的一类缺陷——所以把这个约定钉下来。
func TestPresetHintsShowTheDefaultValue(t *testing.T) {
	// 与 RunSetup 中 Base URL / 模型名两处的取值规则保持一致。
	hintFor := func(defaultValue, genericHint string) string {
		if defaultValue != "" {
			return defaultValue
		}
		return genericHint
	}
	const generic = "generic-example"

	for _, preset := range ProviderPresets() {
		t.Run(preset.Name, func(t *testing.T) {
			if got := hintFor(preset.BaseURL, generic); preset.BaseURL != "" && got != preset.BaseURL {
				t.Errorf("Base URL 提示语 = %q，应显示默认值 %q", got, preset.BaseURL)
			}
			if got := hintFor(preset.DefaultModel, generic); preset.DefaultModel != "" && got != preset.DefaultModel {
				t.Errorf("模型名提示语 = %q，应显示默认值 %q", got, preset.DefaultModel)
			}
			// 没有默认值的预设保持泛例，不能被这条规则误伤。
			if preset.DefaultModel == "" && hintFor(preset.DefaultModel, generic) != generic {
				t.Error("无默认模型的预设应保留泛例提示语")
			}
		})
	}
}
