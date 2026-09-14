package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	lineagev1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/lineage/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
)

// verify reads the graph back.
//
// None of this is needed to write the estate — it is here because the way this
// goes wrong is silent. A missing binding, a schema that was never declared, a
// column edge whose table edge is absent: each of them leaves a valid-looking
// write and an empty result. Reading the lineage back after a sync is the only
// thing that distinguishes "no upstreams" from "upstreams that did not resolve".
//
// Lineage is computed asynchronously from what was written: the SQL is parsed,
// bindings are resolved, the graph is rebuilt. So this polls rather than asking
// once.
func verify(ctx context.Context, api *clients, cfg config, budget time.Duration) {
	dash := dashboardID(dashboards[0].id)

	// The table-level chain. Expect five hops: dashboard <- question <- model <-
	// warehouse tables, plus the filter's direct warehouse edge.
	wantTableNodes := 7
	fmt.Printf("\n-- table lineage upstream of %s\n", dash.GetCustom().GetId())
	table := poll(ctx, budget, func() (*lineagev1.Lineage, bool) {
		lin := getLineage(ctx, api, &lineagev1.GetLineageStartPoint{
			From: &lineagev1.GetLineageStartPoint_Entities{
				Entities: &lineagev1.EntitiesStartPoint{Entities: []*entitiesv1.Identifier{dash}},
			},
		})
		return lin, lin != nil && len(lin.GetNodes()) >= wantTableNodes
	})
	printTableLineage(table)

	// The column-level chain, which is the claim the whole example is making: a
	// field on a dashboard that runs no SQL traces through a question and a
	// model, neither of which the warehouse can address, to the physical column
	// the number came from.
	col := "revenue_by_category.revenue"
	fmt.Printf("\n-- column lineage upstream of %s.%s\n", dash.GetCustom().GetId(), col)
	cll := poll(ctx, budget, func() (*lineagev1.Lineage, bool) {
		lin := getLineage(ctx, api, &lineagev1.GetLineageStartPoint{
			From: &lineagev1.GetLineageStartPoint_EntityColumns{
				EntityColumns: &lineagev1.EntityColumnsStartPoint{
					Id:          dash,
					ColumnNames: []string{col},
				},
			},
		})
		return lin, lin != nil && len(lin.GetColumnDependencies()) > 0
	})
	printColumnLineage(cll, cfg)
}

func getLineage(ctx context.Context, api *clients, from *lineagev1.GetLineageStartPoint) *lineagev1.Lineage {
	depth := int32(10)
	resp, err := api.lineage.GetLineage(ctx, &lineagev1.GetLineageRequest{
		LineageDirection: lineagev1.LineageDirection_LINEAGE_DIRECTION_UPSTREAM,
		StartPoint:       from,
		MaxDepth:         &depth,
	})
	if err != nil {
		fmt.Printf("   (lineage read failed: %v)\n", err)
		return nil
	}
	return resp.GetLineage()
}

func poll(ctx context.Context, budget time.Duration, once func() (*lineagev1.Lineage, bool)) *lineagev1.Lineage {
	deadline := time.Now().Add(budget)
	for {
		lin, done := once()
		if done {
			return lin
		}
		if time.Now().After(deadline) {
			fmt.Printf("   (gave up after %s — the graph may still be building)\n", budget)
			return lin
		}
		select {
		case <-ctx.Done():
			return lin
		case <-time.After(10 * time.Second):
		}
	}
}

func printTableLineage(lin *lineagev1.Lineage) {
	if lin == nil {
		return
	}
	for i, n := range lin.GetNodes() {
		fmt.Printf("   [%d] %-14s %s\n", i, strings.TrimPrefix(n.GetPosition().String(), "NODE_POSITION_"), nodeName(n))
	}
	for _, d := range lin.GetNodeDependencies() {
		fmt.Printf("   %s -> %s\n",
			nodeName(lin.GetNodes()[d.GetSourceNodeIdx()]),
			nodeName(lin.GetNodes()[d.GetTargetNodeIdx()]),
		)
	}
}

func printColumnLineage(lin *lineagev1.Lineage, _ config) {
	if lin == nil {
		return
	}
	for _, n := range lin.GetNodes() {
		if state := n.GetCllDetails().GetCllState(); state != lineagev1.CllState_CLL_STATE_OK &&
			state != lineagev1.CllState_CLL_STATE_UNSPECIFIED {
			// Worth surfacing: RESOLUTION_FAILED on a node is usually a name the
			// definition should have bound and did not.
			fmt.Printf("   %s: %s %v\n", nodeName(n),
				strings.TrimPrefix(state.String(), "CLL_STATE_"), n.GetCllDetails().GetCllMessages())
		}
	}
	for _, d := range lin.GetColumnDependencies() {
		fmt.Printf("   %s.%s -> %s.%s\n",
			nodeName(lin.GetNodes()[d.GetSourceNodeIdx()]), d.GetSourceNodeColumnId(),
			nodeName(lin.GetNodes()[d.GetTargetNodeIdx()]), d.GetTargetNodeColumnId(),
		)
	}
}

// nodeName prefers the identifier shape that says what the node IS. A node
// carries every identifier that resolves to it, so a warehouse table reached
// through a custom entity's binding may list several.
func nodeName(n *lineagev1.LineageNode) string {
	for _, id := range n.GetIds() {
		if c := id.GetCustom(); c != nil {
			return c.GetId()
		}
		if b := id.GetBigqueryTable(); b != nil {
			return fmt.Sprintf("%s.%s.%s", b.GetProject(), b.GetDataset(), b.GetTable())
		}
	}
	if len(n.GetIds()) > 0 {
		return n.GetIds()[0].GetEntityId()
	}
	return "?"
}
