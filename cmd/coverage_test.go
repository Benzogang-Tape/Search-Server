package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestSearchRequest struct {
	Limit      string
	Offset     string
	Query      string
	OrderField string
	OrderBy    string
}

type TestSearchRequestCase struct {
	Request TestSearchRequest
	Error   error
}

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
	errInternalServerError = errors.New("SearchServer fatal error")
	errInvalidOrderField   = errors.New("OrderFeld gender invalid")
	errUnmarshalFailed     = errors.New("cant unpack error json: json: cannot unmarshal string into Go value of type main.SearchErrorResponse")
	errInvalidOrderByParam = errors.New("unknown bad request error: bad order_by param")
	errCantUnpackJSON      = errors.New("cant unpack result json: json: cannot unmarshal string into Go value of type []main.User")
)

var pauseDuration = time.Millisecond
var defaultAccessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNTE2MjM5MDIyfQ.t42p4AHef69Tyyi88U6-p0utZYYrg7mmCGhoAd7Zffs"

var defaultTestCase = TestCase{
	AccessToken: defaultAccessToken,
	Request: &SearchRequest{
		Limit:      12,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	},
	Result: nil,
	Error:  errResponseTimeout,
}

var TestSearchRequests = []TestSearchRequestCase{
	{
		Request: TestSearchRequest{
			Limit:      "",
			Offset:     "",
			Query:      "",
			OrderField: "",
			OrderBy:    "",
		},
		Error: errBadQueryParams,
	},
	{
		Request: TestSearchRequest{
			Limit:      "1",
			Offset:     "",
			Query:      "",
			OrderField: "",
			OrderBy:    "",
		},
		Error: errBadQueryParams,
	},
	{
		Request: TestSearchRequest{
			Limit:      "1",
			Offset:     "0",
			Query:      "",
			OrderField: "",
			OrderBy:    "",
		},
		Error: errBadQueryParams,
	},
	{
		Request: TestSearchRequest{
			Limit:      "-1",
			Offset:     "0",
			Query:      "",
			OrderField: "",
			OrderBy:    "0",
		},
		Error: errBadLimitParam,
	},
	{
		Request: TestSearchRequest{
			Limit:      "1",
			Offset:     "-1",
			Query:      "",
			OrderField: "",
			OrderBy:    "0",
		},
		Error: errBadOffsetParam,
	},
}

var clientTestCases = []TestCase{
	{
		AccessToken: defaultAccessToken,
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
		AccessToken: defaultAccessToken,
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
	{
		AccessToken: defaultAccessToken,
		Request: &SearchRequest{
			Limit:      2,
			Offset:     13,
			Query:      "Aguilar",
			OrderField: "",
			OrderBy:    0,
		},
		Result: &SearchResponse{
			Users:    []User{},
			NextPage: false,
		},
		Error: nil,
	},
	{
		AccessToken: defaultAccessToken,
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
		AccessToken: defaultAccessToken,
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
		AccessToken: defaultAccessToken,
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
		AccessToken: defaultAccessToken,
		Request: &SearchRequest{
			Limit:      1,
			Offset:     0,
			Query:      "",
			OrderField: "",
			OrderBy:    54,
		},
		Result: nil,
		Error:  errInvalidOrderByParam,
	},
}

func (srv *SearchClient) FindUsersSimulator(req TestSearchRequest) (*SearchResponse, error) {
	searcherParams := url.Values{}
	searcherParams.Add("limit", req.Limit)
	searcherParams.Add("offset", req.Offset)
	searcherParams.Add("query", req.Query)
	searcherParams.Add("order_field", req.OrderField)
	searcherParams.Add("order_by", req.OrderBy)

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
		return nil, errors.New(errResp.Error)
	default:
		return &SearchResponse{}, nil
	}
}

func SearchServerSimulator(w http.ResponseWriter, r *http.Request) {
	time.Sleep(pauseDuration)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("Some unnecessary data"); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchServerSimulator2(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("broken json"); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func TestFindUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	for caseNum, item := range clientTestCases {
		cl := &SearchClient{
			AccessToken: item.AccessToken,
			URL:         ts.URL,
		}

		result, err := cl.FindUsers(*item.Request)
		if err != nil {
			assert.Equal(t, item.Error, err, fmt.Sprintf("[%d] Wrong error is returned", caseNum))
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] Wrong response.\nExpected: \n%v\n\nGot: %v", caseNum, item.Result, *result)
		}
	}
}

func TestFindUsersTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerSimulator))
	pauseDuration = time.Second * 2
	defaultTestCase.Error = errResponseTimeout
	cl := &SearchClient{
		AccessToken: defaultTestCase.AccessToken,
		URL:         ts.URL,
	}

	_, err := cl.FindUsers(*defaultTestCase.Request)
	if err != nil {
		assert.Equal(t, defaultTestCase.Error, err, "Wrong error is returned")
	}
	pauseDuration = time.Millisecond
}

func TestFindUsersBrokenJSONResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerSimulator))
	defaultTestCase.Error = errCantUnpackJSON
	cl := &SearchClient{
		AccessToken: defaultTestCase.AccessToken,
		URL:         ts.URL,
	}

	_, err := cl.FindUsers(*defaultTestCase.Request)
	if err != nil {
		assert.Equal(t, defaultTestCase.Error, err, "Wrong error is returned")
	}
}

func TestFindUsersBrokenJSONError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerSimulator2))
	defaultTestCase.Error = errUnmarshalFailed
	cl := &SearchClient{
		AccessToken: defaultTestCase.AccessToken,
		URL:         ts.URL,
	}
	_, err := cl.FindUsers(*defaultTestCase.Request)
	if err != nil {
		assert.Equal(t, defaultTestCase.Error, err, "Wrong error is returned")
	}
}
func TestFindUsersUnknownResponse(t *testing.T) {
	defaultTestCase.Error = errUnknownResponse
	cl := &SearchClient{
		AccessToken: defaultTestCase.AccessToken,
		URL:         "",
	}

	_, err := cl.FindUsers(*defaultTestCase.Request)
	if err != nil {
		assert.Equal(t, defaultTestCase.Error, err, "Wrong error is returned")
	}
}

func TestFindUsersNoDB(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	database = "db/" + database
	defaultTestCase.Error = errInternalServerError
	cl := &SearchClient{
		AccessToken: defaultTestCase.AccessToken,
		URL:         ts.URL,
	}

	_, err := cl.FindUsers(*defaultTestCase.Request)
	if err != nil {
		assert.Equal(t, defaultTestCase.Error, err, "Wrong error is returned")
	}

	database = "dataset.xml"
}

func TestFindUsersBrokenDB(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	database = "broken_dataset.xml"
	defaultTestCase.Error = errInternalServerError
	cl := &SearchClient{
		AccessToken: defaultTestCase.AccessToken,
		URL:         ts.URL,
	}

	_, err := cl.FindUsers(*defaultTestCase.Request)
	if err != nil {
		assert.Equal(t, defaultTestCase.Error, err, "Wrong error is returned")
	}

	database = "dataset.xml"
}

func TestSearchServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	cl := &SearchClient{
		AccessToken: defaultAccessToken,
		URL:         ts.URL,
	}
	for caseNum, item := range TestSearchRequests {
		_, err := cl.FindUsersSimulator(item.Request)
		if err != nil {
			assert.Equal(t, item.Error, err, fmt.Sprintf("[%d] Wrong error is returned", caseNum))
		} else {
			t.Errorf("Error expected, returned nil")
		}
	}
}

func TestSortUsers(t *testing.T) {
	users := []UserClient{
		{
			ID: 0,
		},
		{
			ID: 1,
		},
		{
			ID: 0,
		},
	}
	expectedUsers := []UserClient{
		{
			ID: 0,
		},
		{
			ID: 0,
		},
		{
			ID: 1,
		},
	}
	orderField := "id"
	orderBy := 1

	result := sortUsers(users, orderField, orderBy)
	if !reflect.DeepEqual(expectedUsers, result) {
		t.Errorf("Wrong response.\nExpected: \n%v\n\nGot: %v", expectedUsers, result)
	}
}
