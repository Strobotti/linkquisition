package main_test

import (
	"github.com/strobotti/linkquisition/mock"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strobotti/linkquisition"
	. "github.com/strobotti/linkquisition/plugins/unwrap"
)

func TestUnwrap_ModifyUrl(t *testing.T) {
	mockIoWriter := mock.Writer{
		WriteFunc: func(p []byte) (n int, err error) {
			return len(p), nil
		},
	}
	logger := slog.New(slog.NewTextHandler(mockIoWriter, nil))

	for _, tt := range []struct {
		name            string
		config          map[string]interface{}
		inputUrl        string
		expectedUrl     string
		browserSettings []linkquisition.BrowserSettings
	}{
		{
			name: "Microsoft Teams Defender Safelinks are unwapped",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb&dest=https%3A%2F%2Fteams.microsoft.com%2Fapi%2Fmt%2Fpart%2Femea-03%2Fbeta%2Fatpsafelinks%2Fgeturlreputationsitev2%2F&pc=Cv%252b1z2BmLU9JTROildRi3A9xm64YU8ANInb%252b8ngTDGRd4yd7zmof6UB9e8GJrZhoNyw82Cl6ZrG3YBBiQINgfP2XwQF8KEYw1KV0YRUQBoUYfJ0rChhc7ZyTmruhdrd0ZBjoXUIL4%252bIcXEKyX%252bcfPmSrA2OD7SDEout%252f%252b0690GxXxZQ7%252fdZZfShz5zQhz6UUguNnmnNs9IIwUQ1P00E5AP47cVOxg7M%252bXbARh3DMrPZ8SGUZb%252ffjy3kutqeAShnB4Tl9s1r%252fhiuhKRAE6ZHMzmWsyCi8LJk4FkrAaxhn%252fU%252fOXIH8a2yE49RKJnBZxbeb8Hric0ShgyfDV%252fxGuUzkgR0APorCa8MapNC9yOosor26RKbQily6Kg3ESjTrDNP1%252fryRDfuejy09zOPKl9LhapvQcxxN3%252fFFOm0XUqO9yGr7VizcSdMK1pd20PeIeVkUjFRcxtCx53RGsTbcLsMX%252fYL%252bT%252bVasR%252feDGWZkvfZLQ7UOoqdPcITUArqFHZL9%252fkhnMo1JOHbKT9xVcF6y%252fZPoxY29kKVduOFEMxf8P2ywsv0BKNQgEJ03K%252bus1bTL45m5E32FdmxR57w9M6HPhMMqMMBrmXTGpmsgHo1Ea6wWIZAsQo9IzKVu8JpF9FxczpsyFjLqpUE9kZFMxwN3qlUG%252b68UdA1i9jyGlZf2narRxIWJ3ZerbuXJscIMcJZUOcZktAG6VuPReryltlY01nhf4JyVGsIXHr%252beF8geRW4rq2NB3jHNP18d1jWQdyVaYBBtHlG0%252b8MKJuJbXrYCZYIEPFZ0sfaG4m6qlPmXBN3s20j5jTPG57tFl3Ezsgwp%252b%252fPXTLjdWbfLEdIFxkTvAjKETIBU6fjuKX092UxuvqYrHfvtEBKNW3ASz50entxk3EG5FKcb0YPb%252bpwE4J3qNkDdkTyBWit%252fYma%252bE7j0ib3KWpS3CfdUEqrsRts3V5PdYsDdjG1YUJs92UTZeXpDbs05WrmCK4J8F68yVql%252bAQJIQzabGUqyz9%252bzZhHgi8TMztUEH1SJ3pRfk%252bRc7radrUJknaTWDYQBeQsiENyHMntKX4sykFc%252fDdXl0hu36EODpJQIsggSfNULAl%252b2PwRvojPBwAgVgR36yuED0Z1qsbiTEiEq1leMWZmimOjN8ptV20P%252bPMFJsP1gtQuSXkp3BjMId6p1%252fy5CtNdexxoZ9tZIkPbBSMdHvEwW1PQVpmbWGZj98%252fHMK2bG97gx2S3FlL570XnW3fe39aN6s0carNWpIcqYdSuI3cfU2verCD9VF1eOWWdEwnngnU2h4P%252bx8z8PAm6ZaFpO%252fLhD1zSxKzPeYUvvJdymUcpTT%252bUYNWnxjxlxK7aeCWccsUXo0SUOLz9NxsQUH9VMmQHcnK5Brb0wCRW6QS99Kudqj1BDZ8Wev4RAupHKm%252fI8X52KAKuN54B6fu7mHBL%252b0dTHxz6KulP%252fFfKXFgEkGldnnwfgw4ISRsJ%3B%20expires%3DSun%2C%2025%20Feb%202024%2013%3A02%3A50%20GMT%3B%20path%3D%2F&wau=https%3A%2F%2FEUR02.safelinks.protection.outlook.com%2FGetUrlReputation&si=1708863910422%3B1708863910422%3B48%3Anotes&sd=%7BconvId%3A%2048%3Anotes%2C%20messageId%3A%201708863910422%7D&ce=prod&cv=1415%2F24011826104&ssid=f6342e5f-884d-9e1c-1308-06c74577f23d&ring=ring3_6&clickparams=eyJBcHBOYW1lIjoiVGVhbXMtRGVza3RvcCIsIkFwcFZlcnNpb24iOiIxNDE1LzI0MDExODI2MTA0IiwiSGFzRmVkZXJhdGVkVXNlciI6ZmFsc2V9&bg=%23141414&fg=%23fff&fg2=%237A80EB",
			expectedUrl: "https://github.com/Strobotti/linkquisition",
		},
		{
			name: "Microsoft Teams Defender Safelinks are not unwapped if the URL does not match the rule",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": false,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://www.example.com/path/to/something?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
			expectedUrl: "https://www.example.com/path/to/something?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition",
		},
		{
			name: "Microsoft Teams Defender Safelinks are not unwapped if the unwapped URL would not match any rule browsers rules",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": true,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb&dest=https%3A%2F%2Fteams.microsoft.com%2Fapi%2Fmt%2Fpart%2Femea-03%2Fbeta%2Fatpsafelinks%2Fgeturlreputationsitev2%2F&pc=Cv%252b1z2BmLU9JTROildRi3A9xm64YU8ANInb%252b8ngTDGRd4yd7zmof6UB9e8GJrZhoNyw82Cl6ZrG3YBBiQINgfP2XwQF8KEYw1KV0YRUQBoUYfJ0rChhc7ZyTmruhdrd0ZBjoXUIL4%252bIcXEKyX%252bcfPmSrA2OD7SDEout%252f%252b0690GxXxZQ7%252fdZZfShz5zQhz6UUguNnmnNs9IIwUQ1P00E5AP47cVOxg7M%252bXbARh3DMrPZ8SGUZb%252ffjy3kutqeAShnB4Tl9s1r%252fhiuhKRAE6ZHMzmWsyCi8LJk4FkrAaxhn%252fU%252fOXIH8a2yE49RKJnBZxbeb8Hric0ShgyfDV%252fxGuUzkgR0APorCa8MapNC9yOosor26RKbQily6Kg3ESjTrDNP1%252fryRDfuejy09zOPKl9LhapvQcxxN3%252fFFOm0XUqO9yGr7VizcSdMK1pd20PeIeVkUjFRcxtCx53RGsTbcLsMX%252fYL%252bT%252bVasR%252feDGWZkvfZLQ7UOoqdPcITUArqFHZL9%252fkhnMo1JOHbKT9xVcF6y%252fZPoxY29kKVduOFEMxf8P2ywsv0BKNQgEJ03K%252bus1bTL45m5E32FdmxR57w9M6HPhMMqMMBrmXTGpmsgHo1Ea6wWIZAsQo9IzKVu8JpF9FxczpsyFjLqpUE9kZFMxwN3qlUG%252b68UdA1i9jyGlZf2narRxIWJ3ZerbuXJscIMcJZUOcZktAG6VuPReryltlY01nhf4JyVGsIXHr%252beF8geRW4rq2NB3jHNP18d1jWQdyVaYBBtHlG0%252b8MKJuJbXrYCZYIEPFZ0sfaG4m6qlPmXBN3s20j5jTPG57tFl3Ezsgwp%252b%252fPXTLjdWbfLEdIFxkTvAjKETIBU6fjuKX092UxuvqYrHfvtEBKNW3ASz50entxk3EG5FKcb0YPb%252bpwE4J3qNkDdkTyBWit%252fYma%252bE7j0ib3KWpS3CfdUEqrsRts3V5PdYsDdjG1YUJs92UTZeXpDbs05WrmCK4J8F68yVql%252bAQJIQzabGUqyz9%252bzZhHgi8TMztUEH1SJ3pRfk%252bRc7radrUJknaTWDYQBeQsiENyHMntKX4sykFc%252fDdXl0hu36EODpJQIsggSfNULAl%252b2PwRvojPBwAgVgR36yuED0Z1qsbiTEiEq1leMWZmimOjN8ptV20P%252bPMFJsP1gtQuSXkp3BjMId6p1%252fy5CtNdexxoZ9tZIkPbBSMdHvEwW1PQVpmbWGZj98%252fHMK2bG97gx2S3FlL570XnW3fe39aN6s0carNWpIcqYdSuI3cfU2verCD9VF1eOWWdEwnngnU2h4P%252bx8z8PAm6ZaFpO%252fLhD1zSxKzPeYUvvJdymUcpTT%252bUYNWnxjxlxK7aeCWccsUXo0SUOLz9NxsQUH9VMmQHcnK5Brb0wCRW6QS99Kudqj1BDZ8Wev4RAupHKm%252fI8X52KAKuN54B6fu7mHBL%252b0dTHxz6KulP%252fFfKXFgEkGldnnwfgw4ISRsJ%3B%20expires%3DSun%2C%2025%20Feb%202024%2013%3A02%3A50%20GMT%3B%20path%3D%2F&wau=https%3A%2F%2FEUR02.safelinks.protection.outlook.com%2FGetUrlReputation&si=1708863910422%3B1708863910422%3B48%3Anotes&sd=%7BconvId%3A%2048%3Anotes%2C%20messageId%3A%201708863910422%7D&ce=prod&cv=1415%2F24011826104&ssid=f6342e5f-884d-9e1c-1308-06c74577f23d&ring=ring3_6&clickparams=eyJBcHBOYW1lIjoiVGVhbXMtRGVza3RvcCIsIkFwcFZlcnNpb24iOiIxNDE1LzI0MDExODI2MTA0IiwiSGFzRmVkZXJhdGVkVXNlciI6ZmFsc2V9&bg=%23141414&fg=%23fff&fg2=%237A80EB",
			expectedUrl: "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb&dest=https%3A%2F%2Fteams.microsoft.com%2Fapi%2Fmt%2Fpart%2Femea-03%2Fbeta%2Fatpsafelinks%2Fgeturlreputationsitev2%2F&pc=Cv%252b1z2BmLU9JTROildRi3A9xm64YU8ANInb%252b8ngTDGRd4yd7zmof6UB9e8GJrZhoNyw82Cl6ZrG3YBBiQINgfP2XwQF8KEYw1KV0YRUQBoUYfJ0rChhc7ZyTmruhdrd0ZBjoXUIL4%252bIcXEKyX%252bcfPmSrA2OD7SDEout%252f%252b0690GxXxZQ7%252fdZZfShz5zQhz6UUguNnmnNs9IIwUQ1P00E5AP47cVOxg7M%252bXbARh3DMrPZ8SGUZb%252ffjy3kutqeAShnB4Tl9s1r%252fhiuhKRAE6ZHMzmWsyCi8LJk4FkrAaxhn%252fU%252fOXIH8a2yE49RKJnBZxbeb8Hric0ShgyfDV%252fxGuUzkgR0APorCa8MapNC9yOosor26RKbQily6Kg3ESjTrDNP1%252fryRDfuejy09zOPKl9LhapvQcxxN3%252fFFOm0XUqO9yGr7VizcSdMK1pd20PeIeVkUjFRcxtCx53RGsTbcLsMX%252fYL%252bT%252bVasR%252feDGWZkvfZLQ7UOoqdPcITUArqFHZL9%252fkhnMo1JOHbKT9xVcF6y%252fZPoxY29kKVduOFEMxf8P2ywsv0BKNQgEJ03K%252bus1bTL45m5E32FdmxR57w9M6HPhMMqMMBrmXTGpmsgHo1Ea6wWIZAsQo9IzKVu8JpF9FxczpsyFjLqpUE9kZFMxwN3qlUG%252b68UdA1i9jyGlZf2narRxIWJ3ZerbuXJscIMcJZUOcZktAG6VuPReryltlY01nhf4JyVGsIXHr%252beF8geRW4rq2NB3jHNP18d1jWQdyVaYBBtHlG0%252b8MKJuJbXrYCZYIEPFZ0sfaG4m6qlPmXBN3s20j5jTPG57tFl3Ezsgwp%252b%252fPXTLjdWbfLEdIFxkTvAjKETIBU6fjuKX092UxuvqYrHfvtEBKNW3ASz50entxk3EG5FKcb0YPb%252bpwE4J3qNkDdkTyBWit%252fYma%252bE7j0ib3KWpS3CfdUEqrsRts3V5PdYsDdjG1YUJs92UTZeXpDbs05WrmCK4J8F68yVql%252bAQJIQzabGUqyz9%252bzZhHgi8TMztUEH1SJ3pRfk%252bRc7radrUJknaTWDYQBeQsiENyHMntKX4sykFc%252fDdXl0hu36EODpJQIsggSfNULAl%252b2PwRvojPBwAgVgR36yuED0Z1qsbiTEiEq1leMWZmimOjN8ptV20P%252bPMFJsP1gtQuSXkp3BjMId6p1%252fy5CtNdexxoZ9tZIkPbBSMdHvEwW1PQVpmbWGZj98%252fHMK2bG97gx2S3FlL570XnW3fe39aN6s0carNWpIcqYdSuI3cfU2verCD9VF1eOWWdEwnngnU2h4P%252bx8z8PAm6ZaFpO%252fLhD1zSxKzPeYUvvJdymUcpTT%252bUYNWnxjxlxK7aeCWccsUXo0SUOLz9NxsQUH9VMmQHcnK5Brb0wCRW6QS99Kudqj1BDZ8Wev4RAupHKm%252fI8X52KAKuN54B6fu7mHBL%252b0dTHxz6KulP%252fFfKXFgEkGldnnwfgw4ISRsJ%3B%20expires%3DSun%2C%2025%20Feb%202024%2013%3A02%3A50%20GMT%3B%20path%3D%2F&wau=https%3A%2F%2FEUR02.safelinks.protection.outlook.com%2FGetUrlReputation&si=1708863910422%3B1708863910422%3B48%3Anotes&sd=%7BconvId%3A%2048%3Anotes%2C%20messageId%3A%201708863910422%7D&ce=prod&cv=1415%2F24011826104&ssid=f6342e5f-884d-9e1c-1308-06c74577f23d&ring=ring3_6&clickparams=eyJBcHBOYW1lIjoiVGVhbXMtRGVza3RvcCIsIkFwcFZlcnNpb24iOiIxNDE1LzI0MDExODI2MTA0IiwiSGFzRmVkZXJhdGVkVXNlciI6ZmFsc2V9&bg=%23141414&fg=%23fff&fg2=%237A80EB",
			browserSettings: []linkquisition.BrowserSettings{
				{
					Name: "Test Browser",
					Matches: []linkquisition.BrowserMatch{
						{
							Type:  "domain",
							Value: "example.com",
						},
					},
				},
			},
		},
		{
			name: "Microsoft Teams Defender Safelinks are unwapped if the unwapped URL would match any browsers rule",
			config: map[string]interface{}{
				"requireBrowserMatchToUnwrap": true,
				"rules": []map[string]interface{}{
					{
						"match":     "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
						"parameter": "url",
					},
				},
			},
			inputUrl:    "https://statics.teams.cdn.office.net/evergreen-assets/safelinks/1/atp-safelinks.html?url=https%3A%2F%2Fgithub.com%2FStrobotti%2Flinkquisition&locale=en-gb&dest=https%3A%2F%2Fteams.microsoft.com%2Fapi%2Fmt%2Fpart%2Femea-03%2Fbeta%2Fatpsafelinks%2Fgeturlreputationsitev2%2F&pc=Cv%252b1z2BmLU9JTROildRi3A9xm64YU8ANInb%252b8ngTDGRd4yd7zmof6UB9e8GJrZhoNyw82Cl6ZrG3YBBiQINgfP2XwQF8KEYw1KV0YRUQBoUYfJ0rChhc7ZyTmruhdrd0ZBjoXUIL4%252bIcXEKyX%252bcfPmSrA2OD7SDEout%252f%252b0690GxXxZQ7%252fdZZfShz5zQhz6UUguNnmnNs9IIwUQ1P00E5AP47cVOxg7M%252bXbARh3DMrPZ8SGUZb%252ffjy3kutqeAShnB4Tl9s1r%252fhiuhKRAE6ZHMzmWsyCi8LJk4FkrAaxhn%252fU%252fOXIH8a2yE49RKJnBZxbeb8Hric0ShgyfDV%252fxGuUzkgR0APorCa8MapNC9yOosor26RKbQily6Kg3ESjTrDNP1%252fryRDfuejy09zOPKl9LhapvQcxxN3%252fFFOm0XUqO9yGr7VizcSdMK1pd20PeIeVkUjFRcxtCx53RGsTbcLsMX%252fYL%252bT%252bVasR%252feDGWZkvfZLQ7UOoqdPcITUArqFHZL9%252fkhnMo1JOHbKT9xVcF6y%252fZPoxY29kKVduOFEMxf8P2ywsv0BKNQgEJ03K%252bus1bTL45m5E32FdmxR57w9M6HPhMMqMMBrmXTGpmsgHo1Ea6wWIZAsQo9IzKVu8JpF9FxczpsyFjLqpUE9kZFMxwN3qlUG%252b68UdA1i9jyGlZf2narRxIWJ3ZerbuXJscIMcJZUOcZktAG6VuPReryltlY01nhf4JyVGsIXHr%252beF8geRW4rq2NB3jHNP18d1jWQdyVaYBBtHlG0%252b8MKJuJbXrYCZYIEPFZ0sfaG4m6qlPmXBN3s20j5jTPG57tFl3Ezsgwp%252b%252fPXTLjdWbfLEdIFxkTvAjKETIBU6fjuKX092UxuvqYrHfvtEBKNW3ASz50entxk3EG5FKcb0YPb%252bpwE4J3qNkDdkTyBWit%252fYma%252bE7j0ib3KWpS3CfdUEqrsRts3V5PdYsDdjG1YUJs92UTZeXpDbs05WrmCK4J8F68yVql%252bAQJIQzabGUqyz9%252bzZhHgi8TMztUEH1SJ3pRfk%252bRc7radrUJknaTWDYQBeQsiENyHMntKX4sykFc%252fDdXl0hu36EODpJQIsggSfNULAl%252b2PwRvojPBwAgVgR36yuED0Z1qsbiTEiEq1leMWZmimOjN8ptV20P%252bPMFJsP1gtQuSXkp3BjMId6p1%252fy5CtNdexxoZ9tZIkPbBSMdHvEwW1PQVpmbWGZj98%252fHMK2bG97gx2S3FlL570XnW3fe39aN6s0carNWpIcqYdSuI3cfU2verCD9VF1eOWWdEwnngnU2h4P%252bx8z8PAm6ZaFpO%252fLhD1zSxKzPeYUvvJdymUcpTT%252bUYNWnxjxlxK7aeCWccsUXo0SUOLz9NxsQUH9VMmQHcnK5Brb0wCRW6QS99Kudqj1BDZ8Wev4RAupHKm%252fI8X52KAKuN54B6fu7mHBL%252b0dTHxz6KulP%252fFfKXFgEkGldnnwfgw4ISRsJ%3B%20expires%3DSun%2C%2025%20Feb%202024%2013%3A02%3A50%20GMT%3B%20path%3D%2F&wau=https%3A%2F%2FEUR02.safelinks.protection.outlook.com%2FGetUrlReputation&si=1708863910422%3B1708863910422%3B48%3Anotes&sd=%7BconvId%3A%2048%3Anotes%2C%20messageId%3A%201708863910422%7D&ce=prod&cv=1415%2F24011826104&ssid=f6342e5f-884d-9e1c-1308-06c74577f23d&ring=ring3_6&clickparams=eyJBcHBOYW1lIjoiVGVhbXMtRGVza3RvcCIsIkFwcFZlcnNpb24iOiIxNDE1LzI0MDExODI2MTA0IiwiSGFzRmVkZXJhdGVkVXNlciI6ZmFsc2V9&bg=%23141414&fg=%23fff&fg2=%237A80EB",
			expectedUrl: "https://github.com/Strobotti/linkquisition",
			browserSettings: []linkquisition.BrowserSettings{
				{
					Name: "Test Browser",
					Matches: []linkquisition.BrowserMatch{
						{
							Type:  "domain",
							Value: "github.com",
						},
					},
				},
			},
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				testedPlugin := Plugin
				provider := linkquisition.NewPluginServiceProvider(logger, &linkquisition.Settings{Browsers: tt.browserSettings})
				testedPlugin.Setup(provider, tt.config)

				assert.Equal(t, tt.expectedUrl, testedPlugin.ModifyUrl(tt.inputUrl))
			},
		)
	}
}
