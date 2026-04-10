package cli

import (
	"github.com/Omotolani98/foostash/internal/config"
)

func loadProjectConfig() (*config.ProjectConfig, error) {
	return config.LoadProject(".")
}
