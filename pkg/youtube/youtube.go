package youtube

import "goweb_staging/pkg/settings"

var Youtubeclient YoutubeClient

type YoutubeClient struct {
}

func NewYoutubeClient(app *settings.AppConfig) {

	Youtubeclient = YoutubeClient{}
}

func (c *YoutubeClient) TriggerDiscoverNew() {

}

func (c *YoutubeClient) TriggerDiscoverClassic() {

}
