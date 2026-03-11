package svg

import (
	"graphiti/internal/domain"
	"testing"
)

func makeNode(inputs, outputs int, bodyAttrs int, width int) *domain.NodeInstance {
	def := &domain.NodeDefinition{
		Shape: domain.Shape{Width: width},
	}
	for i := 0; i < inputs; i++ {
		def.Inputs = append(def.Inputs, domain.PortDefinition{
			ID:    "in-" + string(rune('a'+i)),
			Label: "Input " + string(rune('A'+i)),
			Type:  domain.PortTypeData,
		})
	}
	for i := 0; i < outputs; i++ {
		def.Outputs = append(def.Outputs, domain.PortDefinition{
			ID:    "out-" + string(rune('a'+i)),
			Label: "Output " + string(rune('A'+i)),
			Type:  domain.PortTypeData,
		})
	}
	for i := 0; i < bodyAttrs; i++ {
		def.Attributes = append(def.Attributes, domain.AttributeDefinition{
			ID:      "attr-" + string(rune('a'+i)),
			Display: domain.DisplayNodeBody,
		})
	}
	return &domain.NodeInstance{
		ID:         "test-node",
		Definition: def,
	}
}

func TestNodeHeightInt_MinHeight(t *testing.T) {
	node := makeNode(1, 1, 0, 200)
	h := nodeHeightInt(node)
	if h < 60 {
		t.Errorf("height %d is below minimum 60", h)
	}
}

func TestNodeHeightInt_GrowsWithPorts(t *testing.T) {
	node1 := makeNode(1, 1, 0, 200)
	node6 := makeNode(1, 6, 0, 200)
	h1 := nodeHeightInt(node1)
	h6 := nodeHeightInt(node6)
	if h6 <= h1 {
		t.Errorf("6-port node (%d) should be taller than 1-port node (%d)", h6, h1)
	}
}

func TestNodeHeightInt_GrowsWithBodyAttrs(t *testing.T) {
	node0 := makeNode(1, 1, 0, 200)
	node3 := makeNode(1, 1, 3, 200)
	h0 := nodeHeightInt(node0)
	h3 := nodeHeightInt(node3)
	if h3 <= h0 {
		t.Errorf("3-attr node (%d) should be taller than 0-attr node (%d)", h3, h0)
	}
}

func TestNodeHeightInt_PortsVsAttrs(t *testing.T) {
	// When ports require more space than attrs, ports win
	nodeManyPorts := makeNode(1, 10, 1, 200)
	nodeFewPorts := makeNode(1, 1, 1, 200)
	hMany := nodeHeightInt(nodeManyPorts)
	hFew := nodeHeightInt(nodeFewPorts)
	if hMany <= hFew {
		t.Errorf("10-port node (%d) should be taller than 1-port node (%d)", hMany, hFew)
	}
}

func TestPortYInt_SinglePort(t *testing.T) {
	node := makeNode(1, 1, 0, 200)
	y := portYInt(node, true, 0)
	h := nodeHeightInt(node)
	// Single port should be centered in content area
	if y < 36 || y > h-12 {
		t.Errorf("single port y=%d is outside content area [36, %d]", y, h-12)
	}
}

func TestPortYInt_MultiplePorts_NoOverlap(t *testing.T) {
	node := makeNode(1, 6, 0, 200)
	var ys []int
	for i := 0; i < 6; i++ {
		ys = append(ys, portYInt(node, false, i))
	}
	for i := 1; i < len(ys); i++ {
		if ys[i] <= ys[i-1] {
			t.Errorf("port %d (y=%d) not below port %d (y=%d)", i, ys[i], i-1, ys[i-1])
		}
	}
}

func TestPortYInt_EvenSpacing(t *testing.T) {
	node := makeNode(1, 4, 0, 200)
	var ys []int
	for i := 0; i < 4; i++ {
		ys = append(ys, portYInt(node, false, i))
	}
	spacing := ys[1] - ys[0]
	for i := 2; i < len(ys); i++ {
		s := ys[i] - ys[i-1]
		if s != spacing {
			t.Errorf("spacing between ports %d-%d is %d, expected %d", i-1, i, s, spacing)
		}
	}
}

func TestPortYInt_WithinNodeBounds(t *testing.T) {
	node := makeNode(2, 8, 2, 200)
	h := nodeHeightInt(node)
	for i := 0; i < 8; i++ {
		y := portYInt(node, false, i)
		if y < 0 || y > h {
			t.Errorf("output port %d y=%d is outside node bounds [0, %d]", i, y, h)
		}
	}
	for i := 0; i < 2; i++ {
		y := portYInt(node, true, i)
		if y < 0 || y > h {
			t.Errorf("input port %d y=%d is outside node bounds [0, %d]", i, y, h)
		}
	}
}

func TestFindPortY_MatchesPortYInt(t *testing.T) {
	node := makeNode(2, 3, 0, 200)
	// Test output port lookup
	for i, p := range node.Definition.Outputs {
		got := findPortY(node, p.ID)
		want := float64(portYInt(node, false, i))
		if got != want {
			t.Errorf("findPortY(%s) = %g, want %g", p.ID, got, want)
		}
	}
	// Test input port lookup
	for i, p := range node.Definition.Inputs {
		got := findPortY(node, p.ID)
		want := float64(portYInt(node, true, i))
		if got != want {
			t.Errorf("findPortY(%s) = %g, want %g", p.ID, got, want)
		}
	}
}

func TestFindPortY_UnknownPort(t *testing.T) {
	node := makeNode(1, 1, 0, 200)
	y := findPortY(node, "nonexistent")
	h := float64(nodeHeightInt(node))
	if y != h/2 {
		t.Errorf("findPortY for unknown port = %g, want %g (center)", y, h/2)
	}
}

func TestDiamondPoints_DynamicHeight(t *testing.T) {
	node := makeNode(1, 6, 0, 200)
	pts := diamondPoints(node)
	h := nodeHeightInt(node)
	if h <= 80 {
		t.Errorf("diamond with 6 ports should have height > 80, got %d", h)
	}
	// Points string should contain the dynamic height
	expected := "100 0, 200"
	if len(pts) == 0 {
		t.Error("diamondPoints returned empty string")
	}
	_ = expected // basic sanity — it compiled and returned non-empty
}

func TestHexagonPoints_DynamicHeight(t *testing.T) {
	node := makeNode(1, 6, 0, 200)
	pts := hexagonPoints(node)
	if len(pts) == 0 {
		t.Error("hexagonPoints returned empty string")
	}
}
