package prompt

type Renderer interface {
	Render(templateName string, variables map[string]any) (string, error)
}
