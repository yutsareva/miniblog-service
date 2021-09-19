package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	openapi3_routers "github.com/getkin/kin-openapi/routers"
	openapi3_legacy "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/stretchr/testify/suite"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

//go:embed api.yaml
var apiSpec []byte

var ctx = context.Background()

func TestAPI(t *testing.T) {
	suite.Run(t, &APISuite{})
}

type APISuite struct {
	suite.Suite

	client        http.Client
	apiSpecRouter openapi3_routers.Router
}

func (s *APISuite) SetupSuite() {
	srv := CreateServer()
	go func() {
		log.Printf("Start serving on %s", srv.Addr)
		log.Fatal(srv.ListenAndServe())
	}()

	spec, err := openapi3.NewLoader().LoadFromData(apiSpec)
	s.Require().NoError(err)
	s.Require().NoError(spec.Validate(ctx))
	router, err := openapi3_legacy.NewRouter(spec)
	s.Require().NoError(err)
	s.apiSpecRouter = router
	s.client.Transport = s.specValidating(http.DefaultTransport)
}

func (s *APISuite) specValidating(transport http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		log.Println("Send HTTP request:")
		reqBody := s.printReq(req)

		// validate request
		route, params, err := s.apiSpecRouter.FindRoute(req)
		s.Require().NoError(err)
		reqDescriptor := &openapi3filter.RequestValidationInput{
			Request:     req,
			PathParams:  params,
			QueryParams: req.URL.Query(),
			Route:       route,
		}
		s.Require().NoError(openapi3filter.ValidateRequest(ctx, reqDescriptor))

		// do request
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		resp, err := transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		log.Println("Got HTTP response:")
		respBody := s.printResp(resp)

		// Validate response against OpenAPI spec
		s.Require().NoError(openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
			RequestValidationInput: reqDescriptor,
			Status:                 resp.StatusCode,
			Header:                 resp.Header,
			Body:                   io.NopCloser(bytes.NewReader(respBody)),
		}))

		return resp, nil
	})
}

func (s *APISuite) printReq(req *http.Request) []byte {
	body := s.readAll(req.Body)

	req.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(req.Write(os.Stdout))
	fmt.Println()

	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *APISuite) printResp(resp *http.Response) []byte {
	body := s.readAll(resp.Body)

	resp.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(resp.Write(os.Stdout))
	fmt.Println()

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *APISuite) readAll(in io.Reader) []byte {
	if in == nil {
		return nil
	}
	data, err := ioutil.ReadAll(in)
	s.Require().NoError(err)
	return data
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type post struct {
	Id        string    `json:"id"`
	AuthorId  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *APISuite) TestSimple() {
	// when:
	urlPost := "http://localhost:8080/api/v1/posts"
	body := strings.NewReader("{\"text\": \"1234\"}")
	reqPost, _ := http.NewRequest("POST", urlPost, body)
	reqPost.Header.Set("Content-Type", "application/json")
	reqPost.Header.Set("System-Design-User-Id", "12345")
	respPost, errPost := s.client.Do(reqPost)

	// then:
	s.Require().NoError(errPost)
	s.Require().Equal(http.StatusOK, respPost.StatusCode)

	var p post
	json.NewDecoder(respPost.Body).Decode(&p)

	urlGet := "http://localhost:8080/api/v1/posts/" + p.Id
	reqGet, _ := http.NewRequest("GET", urlGet, body)
	//req.Header.Set("Content-Type", "application/json")
	reqGet.Header.Set("System-Design-User-Id", "12345")
	respGet, errGet := s.client.Do(reqGet)

	// then:
	s.Require().NoError(errGet)
	s.Require().Equal(http.StatusOK, respGet.StatusCode)

}
