package pkg

import (
	"goweb_staging/pkg/llm"
	"goweb_staging/pkg/settings"
	"goweb_staging/pkg/youtube"
)

func InitPkg(app *settings.AppConfig) {
	llm.NewLLMClient(app)
	youtube.NewYoutubeClient(app)

}
