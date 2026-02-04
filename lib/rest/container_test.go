package rest

import (
	"fmt"
	"net/http"
	"testing"
)

func CreateWebService1() *WebService {
	ws := new(WebService)
	ws.Path("/v1")
	ws.Route(ws.GET("/users").To(UserList))
	ws.Route(ws.GET("/users/{userId}").To(UserGet))
	ws.Route(ws.POST("/users").To(UserCreate))
	ws.Route(ws.GET("/users/{userId:[0-9]+}").To(UserGet))
	ws.Route(ws.DELETE("/users/{userId}").To(UserDelete))
	ws.Route(ws.PUT("/users/{userId}").To(UserUpdate))
	return ws
}

func CreateWebService2() *WebService {
	ws := new(WebService)
	ws.Path("/api/v2")
	ws.Route(ws.GET("/pods").To(Handle2))
	ws.Route(ws.GET("/pods/{podId}").To(Handle2))
	ws.Route(ws.POST("/pods").To(Handle2))
	ws.Route(ws.GET("/pods/{podId:[0-9]+}").To(Handle2))
	ws.Route(ws.DELETE("/pods/{podId}").To(Handle2))
	ws.Route(ws.PUT("/pods/{podId}").To(Handle2))
	return ws
}

func TestContainer(t *testing.T) {
	container := NewContainer()
	container.Add(CreateWebService1()).Add(CreateWebService2())

	for _, ws := range container.RegisteredWebServices() {
		fmt.Println(ws.RootPath())
	}
}

func UserGet(w http.ResponseWriter, r *http.Request) {
	fmt.Println("UserGet")
}

func UserList(w http.ResponseWriter, r *http.Request) {
	fmt.Println("UserList")
}

func UserCreate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("UserCreate")
}

func UserDelete(w http.ResponseWriter, r *http.Request) {
	fmt.Println("UserDelete")
}

func UserUpdate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("UserUpdate")
}

func Handle2(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle2")
}
