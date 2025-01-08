package almanax

const (
	TwitterWebhookType string = "twitter"
	RSSWebhookType            = "rss"
	AlmanaxWebhookType        = "almanax"
)

type AlmanaxResponse struct {
	Date  string `json:"date"`
	Bonus struct {
		Description string `json:"description"`
		BonusType   struct {
			Name string `json:"name"`
			Id   string `json:"id"`
		} `json:"type"`
	} `json:"bonus"`
	RewardKamas int64 `json:"reward_kamas"`
	Tribute     struct {
		Item struct {
			AnkamaId  int64        `json:"ankama_id"`
			ImageUrls ApiImageUrls `json:"image_urls"`
			Name      string       `json:"name"`
			Subtype   string       `json:"subtype"`
		} `json:"item"`
		Quantity int `json:"quantity"`
	} `json:"tribute"`
}

type AlmanaxBonusListing struct {
	Id   string `json:"id"`   // english-id
	Name string `json:"name"` // translated text
}

type AlmanaxBonusListingMeili struct {
	Id   string `json:"id"`   // meili specific id without utf8 guarantees
	Slug string `json:"slug"` // english-id
	Name string `json:"name"` // translated text
}

type ApiImageUrls struct {
	Icon string `json:"icon"`
	Sd   string `json:"sd,omitempty"`
	Hq   string `json:"hq,omitempty"`
	Hd   string `json:"hd,omitempty"`
}

func RenderImageUrls(urls []string) ApiImageUrls {
	if len(urls) == 0 {
		return ApiImageUrls{}
	}

	var res ApiImageUrls
	res.Icon = urls[0]
	if len(urls) > 1 {
		res.Sd = urls[1]
	}
	if len(urls) > 2 {
		res.Hq = urls[2]
	}
	if len(urls) > 3 {
		res.Hd = urls[3]
	}

	return res
}
