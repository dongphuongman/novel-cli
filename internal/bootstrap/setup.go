package bootstrap

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dongphuongman/novel-cli/internal/rules"
	"github.com/dongphuongman/novel-cli/internal/utils"

	"github.com/dongphuongman/novel-cli/internal/lang"

	"github.com/dongphuongman/novel-cli/internal/i18n"
)

// exampleConfig 是引导后写入 ~/.ainovel/config.example.jsonc 的带注释模板。
// 嵌入文件必须与仓库根目录 config.example.jsonc 保持一致，测试会防止漂移。
//
//go:embed config.example.jsonc
var exampleConfig string

// NeedsSetup 检查是否需要首次引导（全局与项目级配置都不存在时触发）。
func NeedsSetup() bool {
	if p := DefaultConfigPath(); p != "" {
		if _, err := os.Stat(p); err == nil {
			return false
		}
	}
	if _, err := os.Stat(projectConfigPath()); err == nil {
		return false
	}
	return true
}

type setupProvider struct {
	name           string
	label          string
	baseURL        string // 预填的 base_url
	needType       bool   // 自定义代理需要额外问 type 和 base_url
	apiKeyOptional bool   // true 表示 API Key 允许留空

	// providerType 为"地址已知的 OpenAI 兼容网关"钉死协议。
	//
	// 这类预设的名字不在 agentcore 的 knownProviders 里，ProviderType() 会因此
	// 报错；钉死 type 是它们能作为预设存在的前提。空 = 内置 provider，协议由
	// 名称本身隐含（保持 11 个既有预设行为逐字节不变）。
	providerType string

	// defaultModel 预填向导的"模型名称"输入框，并在 /config 新增时种下同名条目。
	defaultModel string

	// routedGateway 标记"每次请求换一个后端模型"的网关。既然实际执行的模型逐次
	// 变化，就无法承诺它支持原生 JSON Schema，因此种下的模型显式写 json_schema:false。
	//
	// 今天这只是把既有行为写明白：provider/openai 指向非官方 base_url 时能力报
	// SupportUnknown，llmcontract 本就回落提示词契约。显式写死是为了让意图留在
	// 配置文件里，也防止上游某天改成对未知端点报 Yes。
	routedGateway bool
}

// ProviderPreset 是首次引导和运行时 /config 共用的 provider 目录项。
type ProviderPreset struct {
	Name           string
	Label          string
	BaseURL        string
	NeedType       bool
	APIKeyOptional bool
	ProviderType   string
	DefaultModel   string
	RoutedGateway  bool
}

var setupProviders = []setupProvider{
	{name: "openrouter", label: "OpenRouter", baseURL: "https://openrouter.ai/api/v1"},
	{name: "anthropic", label: "Anthropic"},
	{name: "gemini", label: "Gemini"},
	{name: "openai", label: "OpenAI"},
	{name: "deepseek", label: "DeepSeek"},
	{name: "qwen", label: "Qwen"},
	{name: "glm", label: "GLM"},
	{name: "grok", label: "Grok"},
	{name: "ollama", label: "Ollama", baseURL: "http://localhost:11434/v1", apiKeyOptional: true},
	// 标签里带地址，一眼看出是"跑在自己机器上的网关"，同时全 ASCII——
	// setupProviders 是包级 var，标签一旦想本地化就会撞上 TestNoInitTimeTranslation。
	{
		name: "freellmapi", label: "FreeLLMAPI (localhost:3001)",
		baseURL:       "http://localhost:3001/v1",
		providerType:  "openai",
		defaultModel:  "auto",
		routedGateway: true,
	},
	{name: "bedrock", label: "Bedrock", apiKeyOptional: true},
	{name: "custom", label: "Custom Proxy", needType: true, apiKeyOptional: true},
}

// ProviderPresets 返回一份可安全修改的预设列表。
func ProviderPresets() []ProviderPreset {
	out := make([]ProviderPreset, 0, len(setupProviders))
	for _, preset := range setupProviders {
		out = append(out, ProviderPreset{
			Name: preset.name, Label: preset.label, BaseURL: preset.baseURL,
			NeedType: preset.needType, APIKeyOptional: preset.apiKeyOptional,
			ProviderType: preset.providerType, DefaultModel: preset.defaultModel,
			RoutedGateway: preset.routedGateway,
		})
	}
	return out
}

// RunSetup 运行首次引导，返回生成的配置。
func RunSetup() (Config, error) {
	// Step 1: 界面语言必须最先问——它决定之后每一句提示用什么语言显示。
	// 这一步的标题刻意三语并列：此刻还没有"用户的语言"这个概念。
	uiLang, err := runUILanguageSelect()
	if err != nil {
		return Config{}, err
	}
	i18n.Use(i18n.Resolve(uiLang))

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).
		Render(i18n.T("setup.intro")))
	fmt.Fprintf(os.Stderr, i18n.T("setup.intro.path"), lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(DefaultConfigPath()))
	fmt.Fprint(os.Stderr, i18n.T("setup.intro.hint"))
	fmt.Fprintln(os.Stderr)
	printStepDone(i18n.T("setup.label.uilang"), i18n.DisplayName(i18n.Resolve(uiLang)))

	// Step 2: 选择 Provider
	sp, err := runProviderSelect()
	if err != nil {
		return Config{}, err
	}

	providerName := sp.name
	var pc ProviderConfig
	// 地址已知的 OpenAI 兼容网关：协议由预设钉死，不问用户。空值对 11 个
	// 既有内置 provider 是 no-op，协议仍由名称隐含。
	pc.Type = sp.providerType
	printStepDone(i18n.T("setup.label.provider"), sp.label)

	// 自定义代理：额外问名称和 API 协议类型
	if sp.needType {
		providerName, err = runTextInput(i18n.T("setup.provider.name"), "my-proxy")
		if err != nil {
			return Config{}, err
		}
		providerType, err := runTypeSelect()
		if err != nil {
			return Config{}, err
		}
		pc.Type = providerType
	}

	// Step 2: 输入 API Key
	var apiKey string
	if sp.apiKeyOptional {
		apiKey, err = runOptionalTextInput(i18n.T("setup.step.apikey.optional"), i18n.T("setup.apikey.emptyhint"))
	} else {
		apiKey, err = runTextInput(i18n.T("setup.step.apikey"), "sk-xxx")
	}
	if err != nil {
		return Config{}, err
	}
	pc.APIKey = apiKey
	if apiKey == "" {
		printStepDone(i18n.T("setup.label.apikey"), i18n.T("common.unset"))
	} else {
		printStepDone(i18n.T("setup.label.apikey"), maskKey(apiKey))
	}

	// Step 3: Base URL（直接回车使用官方默认地址）
	baseDefault := sp.baseURL
	baseHint := i18n.T("setup.baseurl.emptyhint")
	if baseDefault != "" {
		baseHint = baseDefault
	}
	baseURL, err := runTextInputWithDefault(i18n.T("setup.step.baseurl"), baseHint, baseDefault)
	if err != nil {
		return Config{}, err
	}
	pc.BaseURL = baseURL
	if baseURL != "" {
		printStepDone(i18n.T("setup.label.baseurl"), baseURL)
	} else {
		printStepDone(i18n.T("setup.label.baseurl"), i18n.T("common.default"))
	}

	// Step 4: 模型名（必填）。预设可预填一个默认值（网关常用 "auto"）；
	// 复用 Base URL 那一步已有的 WithDefault 输入器，不新造控件。
	//
	// 有默认值时把提示语也换成它——照搬 Base URL 的做法。否则用户看到的是
	// "例如：gpt-4o / …" 这类泛例，根本不知道直接回车会得到 auto（实测发现）。
	modelHint := i18n.T("setup.model.hint")
	if sp.defaultModel != "" {
		modelHint = sp.defaultModel
	}
	modelName, err := runTextInputWithDefault(i18n.T("setup.step.model"), modelHint, sp.defaultModel)
	if err != nil {
		return Config{}, err
	}
	printStepDone(i18n.T("setup.label.model"), modelName)
	pc.Models = []ModelConfig{presetModel(sp, modelName)}

	// Step 5: 创作语言（决定正文语言与随之而来的机械口径，不是界面语言）
	language, err := runLanguageSelect()
	if err != nil {
		return Config{}, err
	}
	printStepDone(i18n.T("setup.label.novellang"), lang.Resolve(language).DisplayName())

	cfg := Config{
		Provider:   providerName,
		ModelName:  modelName,
		Providers:  map[string]ProviderConfig{providerName: pc},
		Roles:      map[string]RoleConfig{},
		Style:      "default",
		Language:   language,
		UILanguage: uiLang,
	}

	// 保存
	path := DefaultConfigPath()
	if err := SaveConfig(path, cfg); err != nil {
		return cfg, fmt.Errorf("save config: %w", err)
	}

	// 生成注释模板
	saveExampleConfig()

	// 全局偏好目录由启动流程（runWithConfig）统一创建，这里仅取路径用于提示
	rulesDir := rules.DefaultHomeRulesDir()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, i18n.T("setup.done.saved"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓"), path)
	fmt.Fprintf(os.Stderr, i18n.T("setup.done.model"), modelName)
	fmt.Fprintln(os.Stderr, i18n.T("setup.done.roles"))
	if rulesDir != "" {
		fmt.Fprintf(os.Stderr, i18n.T("setup.done.rules"), rulesDir)
	}
	fmt.Fprintln(os.Stderr)

	return cfg, nil
}

func saveExampleConfig() {
	dir, err := configDir()
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "config.example.jsonc"), []byte(exampleConfig), 0o644)
}

// printStepDone 打印一步完成的确认行。
func printStepDone(label, value string) {
	fmt.Fprintf(os.Stderr, "  %s %s: %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓"),
		label,
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(value))
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// ---------- TUI 组件 ----------

func runProviderSelect() (setupProvider, error) {
	m := setupSelectModel{
		title: i18n.T("setup.step.provider"),
		items: setupProviders,
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return setupProvider{}, err
	}
	result := final.(setupSelectModel)
	if result.cancelled {
		return setupProvider{}, fmt.Errorf("setup cancelled")
	}
	return result.items[result.cursor], nil
}

// runUILanguageSelect 选界面语言。它在任何本地化文案之前运行，
// 因此选项标签只用各语言自己的名字（都能自我识别），标题三语并列。
func runUILanguageSelect() (string, error) {
	var items []setupProvider
	for _, l := range i18n.Supported() {
		items = append(items, setupProvider{name: string(l), label: i18n.DisplayName(l)})
	}
	m := setupSelectModel{
		title: i18n.T("setup.step.uilang"),
		items: items,
	}
	prog := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := prog.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupSelectModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	return result.items[result.cursor].name, nil
}

// runLanguageSelect 让用户选本书正文语言。
//
// 措辞刻意说"创作语言"而非"语言"：它改变的是**产出物**（正文、章节标题、
// 大纲、摘要）与字数口径，界面文案不受影响。
func runLanguageSelect() (string, error) {
	var items []setupProvider
	for _, p := range lang.Supported() {
		label := p.DisplayName()
		if p.EnglishName() != p.DisplayName() {
			label = i18n.T("wrap.paren", label, p.EnglishName())
		}
		items = append(items, setupProvider{name: string(p.Code()), label: label})
	}
	m := setupSelectModel{
		title: i18n.T("setup.step.novellang"),
		items: items,
	}
	prog := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := prog.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupSelectModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	return result.items[result.cursor].name, nil
}

// 函数而非 var：init 期 i18n.Use 尚未执行，包级 var 会钉死默认语言。
func apiTypeOptions() []setupProvider {
	return []setupProvider{
		{name: "openai", label: i18n.T("setup.apitype.openai")},
		{name: "anthropic", label: i18n.T("setup.apitype.anthropic")},
		{name: "gemini", label: i18n.T("setup.apitype.gemini")},
	}
}

func runTypeSelect() (string, error) {
	m := setupSelectModel{
		title: i18n.T("setup.provider.apitype"),
		items: apiTypeOptions(),
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupSelectModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	return result.items[result.cursor].name, nil
}

func runTextInput(label, placeholder string) (string, error) {
	return runTextInputWithDefault(label, placeholder, "")
}

func runOptionalTextInput(label, placeholder string) (string, error) {
	m := setupInputModel{label: label, placeholder: placeholder, allowEmpty: true}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupInputModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	return utils.CleanInputLine(result.value), nil
}

func runTextInputWithDefault(label, placeholder, defaultValue string) (string, error) {
	m := setupInputModel{label: label, placeholder: placeholder, defaultValue: defaultValue}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupInputModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	if result.value == "" && result.defaultValue != "" {
		return result.defaultValue, nil
	}
	return utils.CleanInputLine(result.value), nil
}

// ---------- 选择器 ----------

var (
	setupCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	setupDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	setupHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	setupInputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
)

type setupSelectModel struct {
	title     string
	items     []setupProvider
	cursor    int
	cancelled bool
}

func (m setupSelectModel) Init() tea.Cmd { return nil }

func (m setupSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m setupSelectModel) View() string {
	var b strings.Builder
	b.WriteString(setupHeaderStyle.Render(m.title))
	b.WriteString("\n\n")
	for i, item := range m.items {
		cursor := "  "
		label := item.label
		if i == m.cursor {
			cursor = setupCursorStyle.Render("❯ ")
			label = setupCursorStyle.Render(label)
		}
		b.WriteString(cursor + label + "\n")
	}
	b.WriteString(setupDimStyle.Render(i18n.T("setup.hint.select")))
	return b.String()
}

// ---------- 文本输入 ----------

type setupInputModel struct {
	label        string
	placeholder  string
	defaultValue string // 直接回车时使用的默认值
	allowEmpty   bool   // 允许直接输入空值
	value        string
	cancelled    bool
}

func (m setupInputModel) Init() tea.Cmd { return nil }

func (m setupInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			if utils.CleanInputLine(m.value) != "" || m.defaultValue != "" || m.allowEmpty {
				return m, tea.Quit
			}
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "backspace":
			if len(m.value) > 0 {
				runes := []rune(m.value)
				m.value = string(runes[:len(runes)-1])
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.value += utils.CleanInputRunes(msg.Runes)
			} else if msg.Type == tea.KeySpace {
				m.value += " "
			}
		}
	}
	return m, nil
}

func (m setupInputModel) View() string {
	var b strings.Builder
	b.WriteString(setupHeaderStyle.Render(m.label))
	b.WriteString("\n\n")
	b.WriteString(setupInputStyle.Render("❯ "))
	if m.value == "" {
		b.WriteString(setupCursorStyle.Render("▌"))
		b.WriteString(setupDimStyle.Render(m.placeholder))
	} else {
		b.WriteString(m.value)
		b.WriteString(setupCursorStyle.Render("▌"))
	}
	b.WriteString(setupDimStyle.Render(i18n.T("setup.hint.input")))
	b.WriteString("\n")
	return b.String()
}

// presetModel 按预设语义构造种子模型条目。
//
// routedGateway 的网关每次请求换后端模型，无法承诺原生 JSON Schema，因此显式
// 写死 json_schema:false——让意图留在用户的配置文件里，而不是依赖能力探测的
// 当前行为（详见 setupProvider.routedGateway 的说明）。
func presetModel(sp setupProvider, name string) ModelConfig {
	m := ModelConfig{Name: name}
	if sp.routedGateway {
		off := false
		m.JSONSchema = &off
	}
	return m
}
