package azure

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	priceTypes "dat067/costestimation/pricing/types"
)

const AZURE_HOST = "https://prices.azure.com/api/retail/prices"

const METER_SPOT = "Spot"
const METER_LOW_PRIORITY = "Low Priority"

type Item struct {
	CurrencyCode         string    `json:"currencyCode"`
	TierMinimumUnits     int       `json:"tierMinimumUnits"`
	RetailPrice          float64   `json:"retailPrice"`
	UnitPrice            float64   `json:"unitPrice"`
	ArmRegionName        string    `json:"armRegionName"`
	Location             string    `json:"location"`
	EffectiveStartDate   time.Time `json:"effectiveStartDate"`
	MeterId              string    `json:"meterId"`
	MeterName            string    `json:"meterName"`
	ProductId            string    `json:"productId"`
	SkuId                string    `json:"skuId"`
	ProductName          string    `json:"productName"`
	SkuName              string    `json:"skuName"`
	ServiceName          string    `json:"serviceName"`
	ServiceId            string    `json:"serviceId"`
	ServiceFamily        string    `json:"serviceFamily"`
	UnitOfMeasure        string    `json:"unitOfMeasure"`
	ItemType             string    `json:"type"`
	IsPrimaryMeterRegion bool      `json:"isPrimaryMeterRegion"`
	ArmSkuName           string    `json:"armSkuName"`
}

type QueryResponse struct {
	BillingCurrency    string `json:"BillingCurrency"`
	CustomerEntityId   string `json:"CustomerEntityId"`
	CustomerEntityType string `json:"CustomerEntityType"`
	Items              []Item `json:"Items"`
}

type Currency uint

const (
	USD Currency = iota
	AUD
	BRL
	CAD
	CHF
	CNY
	DKK
	EUR
	GBP
	INR
	JPY
	KRW
	NOK
	NZD
	RUB
	SEK
	TWD
)

func (c Currency) String() (string, error) {
	switch c {
	case USD:
		return "USD", nil
	case BRL:
		return "BRL", nil
	case CAD:
		return "CAD", nil
	case CHF:
		return "CHF", nil
	case CNY:
		return "CNY", nil
	case DKK:
		return "DKK", nil
	case EUR:
		return "EUR", nil
	case GBP:
		return "GBP", nil
	case INR:
		return "INR", nil
	case JPY:
		return "JPY", nil
	case KRW:
		return "KRW", nil
	case NOK:
		return "NOK", nil
	case NZD:
		return "NZD", nil
	case RUB:
		return "RUB", nil
	case SEK:
		return "SEK", nil
	case TWD:
		return "TWD", nil
	default:
		return "", errors.New(fmt.Sprintf("Invalid currency: %d", c))
	}
}

type QueryFilter struct {
	ArmRegionName string
	Location      string
	MeterId       string
	ProductId     string
	SkuId         string
	ProductName   string
	SkuName       string
	ServiceName   string
	ServiceId     string
	ServiceFamily string
	PriceType     string
	ArmSkuName    string
	CurrencyCode  Currency
}

func (q *QueryFilter) String() (string, error) {
	builder := strings.Builder{}

	currency, err := q.CurrencyCode.String()
	if err != nil {
		return "", err
	}

	params := url.Values{}

	params.Add("currencyCode", currency)

	if q.ArmRegionName != "" {
		builder.WriteString(fmt.Sprintf("armRegionName eq '%s' and ", q.ArmRegionName))
	}
	if q.ArmSkuName != "" {
		builder.WriteString(fmt.Sprintf("armSkuName eq '%s' and ", q.ArmSkuName))
	}
	if q.Location != "" {
		builder.WriteString(fmt.Sprintf("location eq '%s' and ", q.Location))
	}
	if q.MeterId != "" {
		builder.WriteString(fmt.Sprintf("meterId eq '%s' and ", q.MeterId))
	}
	if q.PriceType != "" {
		builder.WriteString(fmt.Sprintf("priceType eq '%s' and ", q.PriceType))
	}
	if q.ProductId != "" {
		builder.WriteString(fmt.Sprintf("productId eq '%s' and ", q.ProductId))
	}
	if q.ProductName != "" {
		builder.WriteString(fmt.Sprintf("productName eq '%s' and ", q.ProductName))
	}
	if q.ServiceFamily != "" {
		builder.WriteString(fmt.Sprintf("serviceFamily eq '%s' and ", q.ServiceFamily))
	}
	if q.ServiceId != "" {
		builder.WriteString(fmt.Sprintf("serviceId eq '%s' and ", q.ServiceId))
	}
	if q.ServiceName != "" {
		builder.WriteString(fmt.Sprintf("serviceName eq '%s' and ", q.ServiceName))
	}
	if q.SkuId != "" {
		builder.WriteString(fmt.Sprintf("skuId eq '%s' and ", q.SkuId))
	}
	if q.SkuName != "" {
		builder.WriteString(fmt.Sprintf("skuName eq '%s'", q.SkuName))
	}

	finalFilterString := builder.String()

	if strings.HasSuffix(builder.String(), " and ") {
		finalFilterString = strings.TrimSuffix(builder.String(), " and ")
	}

	if finalFilterString != "" {
		params.Add("$filter", finalFilterString)
	}

	return params.Encode(), nil
}

type CostApi interface {
	Query(q QueryFilter) (QueryResponse, error)
}

type AzureCostApi struct {
}

/*
 * TODO: Return all pages if number of results are > 100. Maybe include an optional "limit" parameter to limit the amount of returned results?
 */
func (a *AzureCostApi) Query(q QueryFilter) (QueryResponse, error) {
	filterString, err := q.String()

	if err != nil {
		return QueryResponse{}, err
	}

	queryUrl := fmt.Sprintf("%s?%s", AZURE_HOST, filterString)
	res, err := http.Get(queryUrl)

	if err != nil {
		return QueryResponse{}, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return QueryResponse{}, err
	}

	response := QueryResponse{}

	err = json.Unmarshal(body, &response)
	return response, nil
}

func NewApi() CostApi {
	return &AzureCostApi{}
}

func ParseUnit(s string) (priceTypes.Unit, error) {
	trimmedString := strings.ToLower(strings.ReplaceAll(s, " ", ""))

	if trimmedString == "" {
		return priceTypes.Unknown, errors.New(fmt.Sprintf("Invalid input value: '%s'", s))
	}

	switch trimmedString {
	case "1hour":
		return priceTypes.OneHour, nil
	case "1minute":
		return priceTypes.OneMinute, nil
	case "1second":
		return priceTypes.OneSecond, nil
	}

	return priceTypes.Unknown, errors.New(fmt.Sprintf("Invalid input value: '%s'", s))
}

func printPrices(r QueryResponse) {
	fmt.Printf("Currency: %s, customer entity id: %s, customer entity type: %s\n", r.BillingCurrency, r.CustomerEntityId, r.CustomerEntityType)

	for _, item := range r.Items {
		fmt.Printf("\tarmRegionName: %s\n", item.ArmRegionName)
		fmt.Printf("\tarmSkuName: %s\n", item.ArmSkuName)
		fmt.Printf("\tcurrencyCode: %s\n", item.CurrencyCode)
		fmt.Printf("\teffectiveStartDate: %s\n", item.EffectiveStartDate)
		fmt.Printf("\tisPrimaryMeterRegion: %v\n", item.IsPrimaryMeterRegion)
		fmt.Printf("\ttype: %s\n", item.ItemType)
		fmt.Printf("\tlocation: %s\n", item.Location)
		fmt.Printf("\tmeterId: %s\n", item.MeterId)
		fmt.Printf("\tmeterName: %s\n", item.MeterName)
		fmt.Printf("\tproductId: %s\n", item.ProductId)
		fmt.Printf("\tproductName: %s\n", item.ProductName)
		fmt.Printf("\tretailPrice: %f\n", item.RetailPrice)
		fmt.Printf("\tserviceFamily: %s\n", item.ServiceFamily)
		fmt.Printf("\tserviceId: %s\n", item.ServiceId)
		fmt.Printf("\tserviceName: %s\n", item.ServiceName)
		fmt.Printf("\tskuId: %s\n", item.SkuId)
		fmt.Printf("\tskuName: %s\n", item.SkuName)
		fmt.Printf("\ttierMinimumUnits: %d\n", item.TierMinimumUnits)
		fmt.Printf("\tunitOfMeasure: %s\n", item.UnitOfMeasure)
		fmt.Printf("\tunitPrice: %f\n", item.UnitPrice)
		fmt.Printf("\n")
	}

	fmt.Printf("\n")
}
