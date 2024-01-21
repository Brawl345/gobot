package rki

import (
	"encoding/json"
	"time"
)

type (
	Data struct {
		Cases         int     `json:"cases"`
		Deaths        int     `json:"deaths"`
		Recovered     int     `json:"recovered"`
		WeekIncidence float64 `json:"weekIncidence"`
		CasesPer100K  float64 `json:"casesPer100k"`
		CasesPerWeek  int     `json:"casesPerWeek"`
		Delta         struct {
			Cases     int `json:"cases"`
			Deaths    int `json:"deaths"`
			Recovered int `json:"recovered"`
		} `json:"delta"`
	}

	Meta struct {
		LastUpdate time.Time `json:"lastUpdate"`
	}

	Nationwide struct {
		Data
		R struct {
			RValue4Days struct {
				Value float64   `json:"value"`
				Date  time.Time `json:"date"`
			} `json:"rValue4Days"`
			RValue7Days struct {
				Value float64   `json:"value"`
				Date  time.Time `json:"date"`
			} `json:"rValue7Days"`
		} `json:"r"`
		Meta Meta `json:"meta"`
	}

	DistrictResponse struct {
		Districts map[string]District
		Meta      Meta
	}

	District struct {
		Data
		Ags    string `json:"ags"`
		Name   string `json:"name"`
		County string `json:"county"`
		State  string `json:"state"`
	}
)

func (districtResponse *DistrictResponse) UnmarshalJSON(bytes []byte) error {
	response := struct {
		Data map[string]any `json:"data"`
		Meta Meta           `json:"meta"`
	}{}

	err := json.Unmarshal(bytes, &response)
	if err != nil {
		return err
	}

	districtResponse.Meta = response.Meta
	districtResponse.Districts = make(map[string]District, len(response.Data))
	for _, district := range response.Data {
		data, err := json.Marshal(district)
		if err != nil {
			return err
		}

		var ds District
		err = json.Unmarshal(data, &ds)
		if err != nil {
			return err
		}

		districtResponse.Districts[ds.Ags] = ds
	}

	return nil

}
