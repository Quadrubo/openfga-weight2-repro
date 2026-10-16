package main

import (
	"context"
	"fmt"
	"time"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/language/pkg/go/transformer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const model = `model
  schema 1.1
type user
type folder
  relations
    define owner: [user]
    define viewer: [user] or owner
type doc
  relations
    define parent: [folder]
    define viewer: viewer from parent
`

func main() {
	ctx := context.Background()
	conn, _ := grpc.NewClient("localhost:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	c := openfgav1.NewOpenFGAServiceClient(conn)
	st, _ := c.CreateStore(ctx, &openfgav1.CreateStoreRequest{Name: "weight2"})
	m, _ := transformer.TransformDSLToProto(model)
	wm, err := c.WriteAuthorizationModel(ctx, &openfgav1.WriteAuthorizationModelRequest{
		StoreId: st.GetId(), SchemaVersion: m.GetSchemaVersion(), TypeDefinitions: m.GetTypeDefinitions(),
	})
	if err != nil {
		panic(err)
	}
	write := func(keys []*openfgav1.TupleKey) {
		_, err := c.Write(ctx, &openfgav1.WriteRequest{StoreId: st.GetId(), Writes: &openfgav1.WriteRequestWrites{TupleKeys: keys}})
		if err != nil {
			panic(err)
		}
	}
	// 50 docs, each in its own folder; anne is a viewer of every second folder.
	var parents, viewers []*openfgav1.TupleKey
	var checks []*openfgav1.BatchCheckItem
	for i := range 50 {
		parents = append(parents, &openfgav1.TupleKey{Object: fmt.Sprintf("doc:%d", i), Relation: "parent", User: fmt.Sprintf("folder:%d", i)})
		if i%2 == 0 {
			viewers = append(viewers, &openfgav1.TupleKey{Object: fmt.Sprintf("folder:%d", i), Relation: "viewer", User: "user:anne"})
		}
		checks = append(checks, &openfgav1.BatchCheckItem{
			TupleKey:      &openfgav1.CheckRequestTupleKey{Object: fmt.Sprintf("doc:%d", i), Relation: "viewer", User: "user:anne"},
			CorrelationId: fmt.Sprint(i),
		})
	}
	write(parents)
	write(viewers)

	start := time.Now()
	resp, err := c.BatchCheck(ctx, &openfgav1.BatchCheckRequest{StoreId: st.GetId(), AuthorizationModelId: wm.GetAuthorizationModelId(), Checks: checks})
	if err != nil {
		panic(err)
	}
	var allowed, failed int
	var msg string
	for _, r := range resp.GetResult() {
		if r.GetError() != nil {
			failed++
			msg = r.GetError().GetMessage()
		} else if r.GetAllowed() {
			allowed++
		}
	}
	fmt.Printf("BatchCheck took %s: %d allowed, %d errors %s\n", time.Since(start).Round(time.Millisecond), allowed, failed, msg)
}
