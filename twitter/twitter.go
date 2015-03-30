package twitter

import (
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"html"
	"regexp"
)

func FormatTweet(tweet anaconda.Tweet) string {
	prefix := ""
	if tweet.Retweeted {
		tweet = *tweet.RetweetedStatus
		prefix = fmt.Sprintf("RT: @%v", tweet.User.ScreenName)
	}
	output := tweet.Text
	for i := len(tweet.Entities.Urls) - 1; i >= 0; i-- {
		url := tweet.Entities.Urls[i]
		output = output[:url.Indices[0]] + url.Expanded_url + output[url.Indices[1]:]
	}
	output = prefix + output
	output = html.UnescapeString(output)

	users := regexp.MustCompile(`(@(\w){1,15})`)
	hashtag := regexp.MustCompile(`(#\w+)`)

	output = users.ReplaceAllString(output, "\002\00302$1\003\002")
	output = hashtag.ReplaceAllString(output, "\00304$1\003")
	return output
}
