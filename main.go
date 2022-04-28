package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kong/go-kong/kong"
	"golang.org/x/sync/semaphore"
)

const WORKSPACE = "workspace"
const PER_ENTITY_NUM_S = 2
const PER_ENTITY_NUM_R = 2
const PER_ENTITY_NUM_P = 0
const PER_ENTITY_NUM_C = 0
const PER_ENTITY_NUM_U = 0

//15k
//const PER_ENTITY_NUM_S = 2000
//const PER_ENTITY_NUM_R = 1500
//const PER_ENTITY_NUM_P = 0
//const PER_ENTITY_NUM_C = 0
//const PER_ENTITY_NUM_U = 0

//30k
//const PER_ENTITY_NUM_S = 10000
//const PER_ENTITY_NUM_R = 10000
//const PER_ENTITY_NUM_P = 8000
//const PER_ENTITY_NUM_C = 2000
//const PER_ENTITY_NUM_U = 2000

//60k
//const PER_ENTITY_NUM_S = 20000
//const PER_ENTITY_NUM_R = 20000
//const PER_ENTITY_NUM_P = 16000
//const PER_ENTITY_NUM_C = 4000
//const PER_ENTITY_NUM_U = 4000

//85k
//const PER_ENTITY_NUM_S = 25000
//const PER_ENTITY_NUM_R = 25000
//const PER_ENTITY_NUM_P = 25000
//const PER_ENTITY_NUM_C = 5000
//const PER_ENTITY_NUM_U = 5000

func init() {
	flag.Parse()
}

var (
	adminAPI      = flag.String("admin-api", "http://127.0.0.1:8001", "Kong CP Admin API")
	mode          = flag.String("mode", "token", "Auth mode")
	auth          = flag.String("auth", "kong_admin:kong", "Basic auth")
	workspace_NUM = flag.Int("workspace-num", 1, "Workspace number to be generated. (default is 1)")
	route_NUM     = flag.Int("route-number-per-service", 1, "route to be generated per service. (default is 1)")
)

func createFile(path string) {
	// detect if the file already exists
	var _, err = os.Stat(path)

	// create a new file if it doesn't already exist
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			panic(err)
		}
		defer file.Close()
	}

	fmt.Println("==> file created successfully", path)
}

type AddHeaderTransport struct {
	T http.RoundTripper
}

func (adt *AddHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	req.Header.Add("Kong-Admin-Token", "admin")
	return adt.T.RoundTrip(req)
}

func NewAddHeaderTransport(T http.RoundTripper) *AddHeaderTransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &AddHeaderTransport{T}
}

type entities struct {
	svcs    []kong.Service
	rts     []kong.Route
	pgns    []kong.Plugin
	csmrs   []kong.Consumer
	upstrms []kong.Upstream
}

func benchServices(client *kong.Client, entities *entities) {
	{
		weight := semaphore.NewWeighted(1)
		start := time.Now()
		wg := new(sync.WaitGroup)
		for i := 0; i < PER_ENTITY_NUM_S; i++ {
			wg.Add(1)
			weight.Acquire(context.Background(), 1)
			j := i
			go func(i int) {
				defer wg.Done()
				defer weight.Release(1)
				svc, err := client.Services.Create(context.Background(), &kong.Service{
					Name: kong.String(fmt.Sprintf("%s-svc-%v", client.Workspace(), i)),
					Host: kong.String(fmt.Sprintf("%s-%v.test.com", client.Workspace(), i)),
				})
				if err != nil {
					log.Println(err)
				}
				entities.svcs = append(entities.svcs, *svc)
			}(j)
		}
		wg.Wait()
		duration := time.Since(start)
		log.Println(fmt.Sprintf("Services: created %v entities in %s, TPS: %v", PER_ENTITY_NUM_S, duration.String(), PER_ENTITY_NUM_S/duration.Seconds()))
	}
	fmt.Println("Print Services")
	path := "./ServiceID_list.csv"
	createFile(path)
	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// write data to file
	for i, _ := range entities.svcs {
		data := *entities.svcs[i].ID
		_, err = file.WriteString(data + "\n")
		if err != nil {
			panic(err)
		}
	}
	err = file.Sync()
	if err != nil {
		panic(err)
	}

}

func benchRoutes(client *kong.Client, entities *entities) {
	{
		weight := semaphore.NewWeighted(1)
		start := time.Now()
		wg := new(sync.WaitGroup)
		for i := 0; i < PER_ENTITY_NUM_R; i++ {
			wg.Add(1)
			weight.Acquire(context.Background(), 1)
			j := i
			go func(i int) {
				defer wg.Done()
				defer weight.Release(1)

				for k := 0; k < *route_NUM; k++ {
					rt, err := client.Routes.CreateInService(context.Background(), entities.svcs[i].ID, &kong.Route{
						Name:  kong.String(fmt.Sprintf("%s-route-%v-%v", client.Workspace(), i, k)),
						Paths: []*string{kong.String(fmt.Sprintf("/%s%v%vtest", client.Workspace(), i, k))},
					})
					if err != nil {
						log.Println(err)
					}
					entities.rts = append(entities.rts, *rt)
				}
			}(j)
		}
		wg.Wait()
		duration := time.Since(start)
		log.Println(fmt.Sprintf("rt: created %v entities in %s, TPS: %v", PER_ENTITY_NUM_R**route_NUM, duration.String(), PER_ENTITY_NUM_R/duration.Seconds()))
	}
	fmt.Println("Print Routes")
	fmt.Println("")
	path := "./RoutesID_list.csv"
	createFile(path)
	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// write data to file
	for i, _ := range entities.rts {
		data := *entities.rts[i].ID
		_, err = file.WriteString(data + "\n")
		if err != nil {
			panic(err)
		}
	}
	err = file.Sync()
	if err != nil {
		panic(err)
	}

}

func benchPlugins(client *kong.Client, entities *entities) {
	{
		weight := semaphore.NewWeighted(1)
		start := time.Now()
		wg := new(sync.WaitGroup)
		for i := 0; i < PER_ENTITY_NUM_P; i++ {
			wg.Add(1)
			weight.Acquire(context.Background(), 1)
			j := i
			go func(i int) {
				defer wg.Done()
				defer weight.Release(1)
				pgn, err := client.Plugins.Create(context.Background(), &kong.Plugin{
					Name:    kong.String("key-auth"),
					Service: &entities.svcs[i],
				})
				if err != nil {
					log.Println(err)
				}
				entities.pgns = append(entities.pgns, *pgn)
			}(j)
		}
		wg.Wait()
		duration := time.Since(start)
		log.Println(fmt.Sprintf("rt: created %v entities in %s, TPS: %v", PER_ENTITY_NUM_P, duration.String(), PER_ENTITY_NUM_P/duration.Seconds()))
	}
	fmt.Println("Print Plugins")
	fmt.Println("")
	path := "./PluginsID_list.csv"
	createFile(path)
	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// write data to file
	for i, _ := range entities.pgns {
		data := *entities.pgns[i].ID
		_, err = file.WriteString(data + "\n")
		if err != nil {
			panic(err)
		}
	}
	err = file.Sync()
	if err != nil {
		panic(err)
	}

}

func benchConsumers(client *kong.Client, entities *entities) {
	{
		weight := semaphore.NewWeighted(1)
		start := time.Now()
		wg := new(sync.WaitGroup)
		for i := 0; i < PER_ENTITY_NUM_C; i++ {
			wg.Add(1)
			weight.Acquire(context.Background(), 1)
			j := i
			go func(i int) {
				defer wg.Done()
				defer weight.Release(1)
				csmr, err := client.Consumers.Create(context.Background(), &kong.Consumer{
					Username: kong.String(fmt.Sprintf("csmr-%v", i)),
				})
				if err != nil {
					log.Println(err)
				}
				entities.csmrs = append(entities.csmrs, *csmr)
			}(j)
		}
		wg.Wait()
		duration := time.Since(start)
		log.Println(fmt.Sprintf("rt: created %v entities in %s, TPS: %v", PER_ENTITY_NUM_C, duration.String(), PER_ENTITY_NUM_C/duration.Seconds()))
	}
	fmt.Println("Print consumers!")
	fmt.Println("")
	path := "./ConsumersID_list.csv"
	createFile(path)
	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// write data to file
	for i, _ := range entities.csmrs {
		data := *entities.csmrs[i].ID
		_, err = file.WriteString(data + "\n")
		if err != nil {
			panic(err)
		}
	}
	err = file.Sync()
	if err != nil {
		panic(err)
	}

}

func benchUpstreams(client *kong.Client, entities *entities) {
	{
		weight := semaphore.NewWeighted(1)
		start := time.Now()
		wg := new(sync.WaitGroup)
		for i := 0; i < PER_ENTITY_NUM_U; i++ {
			wg.Add(1)
			weight.Acquire(context.Background(), 1)
			j := i
			go func(i int) {
				defer wg.Done()
				defer weight.Release(1)
				upstrm, err := client.Upstreams.Create(context.Background(), &kong.Upstream{
					Name: kong.String(fmt.Sprintf("upstrm-%v", i)),
				})
				if err != nil {
					log.Println(err)
				}
				entities.upstrms = append(entities.upstrms, *upstrm)
			}(j)
		}
		wg.Wait()
		duration := time.Since(start)
		log.Println(fmt.Sprintf("rt: created %v entities in %s, TPS: %v", PER_ENTITY_NUM_U, duration.String(), PER_ENTITY_NUM_U/duration.Seconds()))
	}
	fmt.Println("Print Upstreams!")
	fmt.Println("")
	path := "./UpstreamsID_list.csv"
	createFile(path)
	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// write data to file
	for i, _ := range entities.upstrms {
		data := *entities.upstrms[i].ID
		_, err = file.WriteString(data + "\n")
		if err != nil {
			panic(err)
		}
	}
	err = file.Sync()
	if err != nil {
		panic(err)
	}

}

func main() {

	//	entities := &entities{}

	client := &http.Client{
		Transport: NewAddHeaderTransport(nil),
	}

	kc, err := kong.NewClient(adminAPI, client)
	if err != nil {
		log.Fatal(err)
	}

	//	for i, _ := range *workspace_NUM {
	for i := 1; i <= *workspace_NUM; i++ {
		_, err = kc.Workspaces.Create(context.Background(), &kong.Workspace{
			Name: kong.String(WORKSPACE + strconv.Itoa(i)),
		})
		if err != nil {
			log.Println(err)
		}
	}

	for i := 1; i <= *workspace_NUM; i++ {
		entities := &entities{}
		kc.SetWorkspace(WORKSPACE + strconv.Itoa(i))
		_, err = kc.Status(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		benchServices(kc, entities)
		benchRoutes(kc, entities)
		//		benchPlugins(kc, entities)
		//		benchConsumers(kc, entities)
		//		benchUpstreams(kc, entities)
	}

}
