package twitter

type PrivMsg struct {
	Target, Text string
}

type Twitter_conf struct {
	Channels       []string `json:"channels"`
	BotHost        string   `json:"bot_host"`
	ConsumerKey    string   `json:"consumer_key"`
	ConsumerSecret string   `json:"consumer_secret"`
	AccessKey      string   `json:"access_key"`
	AccessSecret   string   `json:"access_secret"`
}
