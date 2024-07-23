package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

type TestCase struct {
	AccessToken string
	Request     *SearchRequest
	Result      *SearchResponse
	Error       error
}

var (
	errNegativeLimitValue  = errors.New("limit must be > 0")
	errNegativeOffsetValue = errors.New("offset must be > 0")
	errResponseTimeout     = errors.New("timeout for limit=13&offset=0&order_by=0&order_field=&query=")
	errUnknownResponse     = errors.New("unknown error Get \"?limit=13&offset=0&order_by=0&order_field=&query=\": unsupported protocol scheme \"\"")
	errBadAccessToken      = errors.New("bad AccessToken")
	errInternalServerError = errors.New("SearchServer fatal error")
	errInvalidOrderField   = errors.New("OrderFeld gender invalid")
	errUnmarshalFailed     = errors.New("cant unpack error json: invalid character ':' after top-level value")
	errInvalidOffsetParam  = errors.New("unknown bad request error: bad offset param")
	errCantUnpackJSON      = errors.New("cant unpack result json: json: cannot unmarshal string into Go value of type []main.User")
	errInvalidQueryParams  = errors.New("bad query params")
)

func (srv *SearchClient) FindUsersSimulator() (*SearchResponse, error) {
	searcherParams := url.Values{}
	searcherParams.Add("key", "value")
	searcherReq, err := http.NewRequest("GET", srv.URL+"?"+searcherParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("error in FindUsersSimulator func: %s", err)
	}
	searcherReq.Header.Add("AccessToken", srv.AccessToken)
	resp, err := client.Do(searcherReq)
	if err != nil {
		return nil, fmt.Errorf("error in FindUsersSimulator func: %s", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}
	switch resp.StatusCode {
	case http.StatusBadRequest:
		errResp := SearchErrorResponse{}
		err := json.Unmarshal(body, &errResp)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %s", err)
		}
		return nil, fmt.Errorf(errResp.Error)
	default:
		return &SearchResponse{}, nil
	}
}

func SearchServerSimulator(w http.ResponseWriter, r *http.Request) {
	time.Sleep(pauseDuration)
	enc := json.NewEncoder(w)
	if err := enc.Encode("Some unnecessary data"); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func TestFindUsers0(t *testing.T) {
	clientTestCases := []TestCase{
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      -54,
				Offset:     1,
				Query:      "",
				OrderField: "age",
				OrderBy:    -1,
			},
			Result: nil,
			Error:  errNegativeLimitValue,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     -1,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errNegativeOffsetValue,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "",
				OrderField: "age",
				OrderBy:    -1,
			},
			Result: &SearchResponse{
				Users: []User{
					{
						ID:     32,
						Name:   "Christy Knapp",
						Age:    40,
						About:  "Incididunt culpa dolore laborum cupidatat consequat. Aliquip cupidatat pariatur sit consectetur laboris labore anim labore. Est sint ut ipsum dolor ipsum nisi tempor in tempor aliqua. Aliquip labore cillum est consequat anim officia non reprehenderit ex duis elit. Amet aliqua eu ad velit incididunt ad ut magna. Culpa dolore qui anim consequat commodo aute.",
						Gender: "female",
					},
				},
				NextPage: true,
			},
			Error: nil,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Hall",
				OrderField: "name",
				OrderBy:    1,
			},
			Result: &SearchResponse{
				Users: []User{
					{
						ID:     18,
						Name:   "Terrell Hall",
						Age:    27,
						About:  "Ut nostrud est est elit incididunt consequat sunt ut aliqua sunt sunt. Quis consectetur amet occaecat nostrud duis. Fugiat in irure consequat laborum ipsum tempor non deserunt laboris id ullamco cupidatat sit. Officia cupidatat aliqua veniam et ipsum labore eu do aliquip elit cillum. Labore culpa exercitation sint sint.",
						Gender: "male",
					},
				},
				NextPage: false,
			},
			Error: nil,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Dillard",
				OrderField: "",
				OrderBy:    -1,
			},
			Result: &SearchResponse{
				Users: []User{
					{
						ID:     3,
						Name:   "Everett Dillard",
						Age:    27,
						About:  "Sint eu id sint irure officia amet cillum. Amet consectetur enim mollit culpa laborum ipsum adipisicing est laboris. Adipisicing fugiat esse dolore aliquip quis laborum aliquip dolore. Pariatur do elit eu nostrud occaecat.",
						Gender: "male",
					},
				},
				NextPage: true,
			},
			Error: nil,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "hender",
				OrderField: "id",
				OrderBy:    -1,
			},
			Result: &SearchResponse{
				Users: []User{
					{
						ID:     33,
						Name:   "Twila Snow",
						Age:    36,
						About:  "Sint non sunt adipisicing sit laborum cillum magna nisi exercitation. Dolore officia esse dolore officia ea adipisicing amet ea nostrud elit cupidatat laboris. Proident culpa ullamco aute incididunt aute. Laboris et nulla incididunt consequat pariatur enim dolor incididunt adipisicing enim fugiat tempor ullamco. Amet est ullamco officia consectetur cupidatat non sunt laborum nisi in ex. Quis labore quis ipsum est nisi ex officia reprehenderit ad adipisicing fugiat. Labore fugiat ea dolore exercitation sint duis aliqua.",
						Gender: "female",
					},
				},
				NextPage: true,
			},
			Error: nil,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      228,
				Offset:     0,
				Query:      "Dillard",
				OrderField: "name",
				OrderBy:    1,
			},
			Result: &SearchResponse{
				Users: []User{
					{
						ID:     17,
						Name:   "Dillard Mccoy",
						Age:    36,
						About:  "Laborum voluptate sit ipsum tempor dolore. Adipisicing reprehenderit minim aliqua est. Consectetur enim deserunt incididunt elit non consectetur nisi esse ut dolore officia do ipsum.",
						Gender: "male",
					},
					{
						ID:     3,
						Name:   "Everett Dillard",
						Age:    27,
						About:  "Sint eu id sint irure officia amet cillum. Amet consectetur enim mollit culpa laborum ipsum adipisicing est laboris. Adipisicing fugiat esse dolore aliquip quis laborum aliquip dolore. Pariatur do elit eu nostrud occaecat.",
						Gender: "male",
					},
				},
				NextPage: false,
			},
			Error: nil,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	for caseNum, item := range clientTestCases {
		cl := &SearchClient{
			AccessToken: item.AccessToken,
			URL:         ts.URL,
		}

		result, err := cl.FindUsers(*item.Request)
		if err != nil {
			assert.Equal(t, item.Error, err, "wrong error is returned")
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong response.\nExpected: \n%v\n\nGot: %v", caseNum, item.Result, *result)
		}
	}
}

func TestFindUsers1(t *testing.T) {
	clientTestCases := []TestCase{
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errResponseTimeout,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errCantUnpackJSON,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errUnknownResponse,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServerSimulator))

	for caseNum, item := range clientTestCases {
		cl := &SearchClient{
			AccessToken: item.AccessToken,
			URL:         ts.URL,
		}
		switch caseNum {
		case 1:
			pauseDuration = time.Millisecond
		case 2:
			cl.URL = ""
		}
		_, err := cl.FindUsers(*item.Request)
		if err != nil {
			assert.Equal(t, item.Error, err, "wrong error is returned")
		}
	}
}

func TestFindUsers2(t *testing.T) {
	clientTestCases := []TestCase{
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errInternalServerError,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errInternalServerError,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "gender",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errUnmarshalFailed,
		},
		{
			AccessToken: "",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errBadAccessToken,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      12,
				Offset:     0,
				Query:      "",
				OrderField: "gender",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errInvalidOrderField,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      20,
				Offset:     13,
				Query:      "Aguilar",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errInvalidOffsetParam,
		},
		{
			AccessToken: "54",
			Request: &SearchRequest{
				Limit:      2,
				Offset:     13,
				Query:      "Aguilar",
				OrderField: "",
				OrderBy:    0,
			},
			Result: nil,
			Error:  errInvalidOffsetParam,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range clientTestCases {
		cl := &SearchClient{
			AccessToken: item.AccessToken,
			URL:         ts.URL,
		}
		switch caseNum {
		case 0:
			database = "db/" + database
		case 1:
			database = "broken_dataset.xml"
		case 2:
			badOrderFieldMessage = badOrderFieldMessage[1:]
		}
		_, err := cl.FindUsers(*item.Request)
		if err != nil {
			assert.Equal(t, item.Error, err, "wrong error is returned")
		}
		database = "dataset.xml"
		badOrderFieldMessage = "{" + "\"error\": " + "\"" + ErrorBadOrderField + "\"" + "}"
	}
}

func TestSearchServer(t *testing.T) {
	serverTestCases := []TestCase{
		{
			AccessToken: "54",
			Request:     nil,
			Result:      nil,
			Error:       errInvalidQueryParams,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	for _, item := range serverTestCases {
		cl := &SearchClient{
			AccessToken: item.AccessToken,
			URL:         ts.URL,
		}

		_, err := cl.FindUsersSimulator()
		if err != nil {
			assert.Equal(t, item.Error, err, "wrong error is returned")
		} else {
			t.Errorf("Error expected, returned nil")
		}
	}
}
