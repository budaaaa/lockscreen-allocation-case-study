package mockpartner_test

import (
	"context"
	"testing"

	"case-study-poc/allocation"
	"case-study-poc/mockpartner"
)

func TestPartnerHonorsRequestedItemCount(t *testing.T) {
	partner := mockpartner.MockPartner{PartnerName: "mock", Items: []allocation.Item{{ID: "one"}, {ID: "two"}}}
	items, err := partner.Fetch(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "one" {
		t.Fatalf("want one item, got %+v", items)
	}
}
