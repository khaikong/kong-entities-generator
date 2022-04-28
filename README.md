## kong-entities-generator

---

The main goal for this tool is to create dummy entities in Kong ASAP! 

### To install

```

git clone https://github.com/khaikong/kong-entities-generator.git
cd kong-entities-generator
go mod init

```

### Creating entities

To create dummy entities, change below parameter in `main.go`

```

// To set number of entities 
const PER_ENTITY_NUM_S = 2
const PER_ENTITY_NUM_R = 2
const PER_ENTITY_NUM_P = 0
const PER_ENTITY_NUM_C = 0
const PER_ENTITY_NUM_U = 0

// To set which entities to be created. 
// Please comment out unwanted functions in `main()` that will create the entities.
// Below example will only create Services and Routes.

		benchServices(kc, entities)
		benchRoutes(kc, entities)
		//		benchPlugins(kc, entities)
		//		benchConsumers(kc, entities)
		//		benchUpstreams(kc, entities)


```

run below command
```
go run main.go -admin-api=http://10.0.0.4:8001 -auth=kong_admin:kong
```

### Available features

#### Create multiple workspaces with set number of entities

Use `-workspace-num` flag
```
# commmand below will create 3 workspaces with set number of entities

go run main.go -admin-api=http://10.0.0.4:8001 -auth=kong_admin:kong -workspace-num 3

```

#### Create multiple routes in one service
Use `route-number-per-service` flag

```
# commmand below will create 15 routes for each service set

go run main.go -admin-api=http://10.0.0.4:8001 -auth=kong_admin:kong -route-number-per-service 15

```
