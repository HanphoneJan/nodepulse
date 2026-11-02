package v2raya

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginTouchAndConnect(t *testing.T) {
	var connectBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["username"] != "u" || body["password"] != "p" {
				http.Error(w, "bad creds", http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"code":200,"message":"ok","data":{"token":"tok"}}`)
		case "/api/touch":
			if r.Header.Get("Authorization") != "Bearer tok" {
				http.Error(w, "no auth", http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"code":200,"data":{"running":true,"touch":{"servers":[{"id":1,"_type":"server","name":"Direct","address":"1.2.3.4:443","net":"ss","pingLatency":"20ms"}],"subscriptions":[{"id":2,"remarks":"Sub","servers":[{"id":1,"_type":"subscriptionServer","name":"Sub Node","address":"5.6.7.8:443","net":"vless","pingLatency":"80ms"}]}],"connectedServer":[{"_type":"subscriptionServer","id":1,"sub":2}]}}}`)
		case "/api/connection":
			_ = json.NewDecoder(r.Body).Decode(&connectBody)
			fmt.Fprint(w, `{"code":200,"message":"ok"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.Login(context.Background(), "u", "p"); err != nil {
		t.Fatal(err)
	}
	if c.Token != "tok" {
		t.Fatalf("token = %q", c.Token)
	}
	touch, err := c.Touch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	nodes := touch.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(nodes))
	}
	n, ok := touch.FindNode("sub node")
	if !ok || n.Name != "Sub Node" {
		t.Fatalf("FindNode = %+v, %v", n, ok)
	}
	if err := c.Connect(context.Background(), n.Ref); err != nil {
		t.Fatal(err)
	}
	if connectBody["_type"] != "subscriptionServer" || connectBody["sub"].(float64) != 2 {
		t.Fatalf("connect body = %#v", connectBody)
	}
}

func TestErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code":500,"message":"boom"}`)
	}))
	defer srv.Close()
	c := New(srv.URL)
	if _, err := c.Touch(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
}
