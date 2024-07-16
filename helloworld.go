package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/qlik-oss/enigma-go/v4"
)

func main() {
	tenant := "xxxx.yy.qlikcloud.com"
	appId := "72a3da4b-1093-4c4c-840d-1ee44fbcbb91"
	qcsApiKey := "eyJhbGci...."
	headers := make(http.Header, 1)
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", qcsApiKey))
	ctx := context.Background()
	global, err := enigma.Dialer{}.Dial(ctx, fmt.Sprintf("wss://%s/app/%s", tenant, appId), headers)
	if err != nil {
		panic(err)
	}

	app, _ := global.OpenDoc(ctx, appId, "", "", "", false)
	var statename = "$"
	app.ClearAll(ctx, false, statename)
	field, _ := app.GetField(ctx, "支店名", statename)
	var fvals = []*enigma.FieldValue{{Text: "関東支店"}, {Text: "関西支店"}}
	select_ok, err := field.SelectValues(ctx, fvals, false, false)
	if !select_ok && err != nil {
		panic(err)
	}

	listobject_def := enigma.GenericObjectProperties{
		Info: &enigma.NxInfo{
			Type: "my-list-object",
		},
		ListObjectDef: &enigma.ListObjectDef{
			Def: &enigma.NxInlineDimensionDef{
				FieldDefs:     []string{"支店名"},
				FieldLabels:   []string{"支店名"},
				SortCriterias: []*enigma.SortCriteria{{SortByLoadOrder: 1}},
			},
			FrequencyMode:    "NX_FREQUENCY_VALUE",
			ShowAlternatives: true,
		},
	}
	lo_hypercube, _ := app.CreateSessionObject(ctx, &listobject_def)
	lo_layout, _ := lo_hypercube.GetLayout(ctx)
	lo_width := lo_layout.ListObject.Size.Cx
	lo_height := 1
	if lo_width != 0 {
		lo_height = int(math.Floor(10000 / float64(lo_width)))
	}
	lo_layout.ListObject.DataPages = []*enigma.NxDataPage{}
	getAllList(ctx, lo_hypercube, lo_layout, lo_width, lo_height, 0)
	renderingList(lo_layout)
	app.DestroySessionObject(ctx, lo_hypercube.GenericId)

	hypercube_def := enigma.GenericObjectProperties{
		Info: &enigma.NxInfo{
			Type: "my-straight-hypercube",
		},
		HyperCubeDef: &enigma.HyperCubeDef{
			Dimensions: []*enigma.NxDimension{{
				Def: &enigma.NxInlineDimensionDef{
					FieldDefs:   []string{"営業員名"},
					FieldLabels: []string{"営業員名"},
				},
				NullSuppression: true,
			}},
			Measures: []*enigma.NxMeasure{{
				Def: &enigma.NxInlineMeasureDef{
					Def:       "Sum([販売価格])",
					Label:     "実績",
					NumFormat: &enigma.FieldAttributes{Type: "MONEY", UseThou: 1, Thou: ","},
				},
				SortBy: &enigma.SortCriteria{
					SortByState:      0,
					SortByFrequency:  0,
					SortByNumeric:    -1, // ソート: 0=無し, 1=昇順, -1=降順
					SortByAscii:      0,
					SortByLoadOrder:  0,
					SortByExpression: 0,
					Expression:       &enigma.ValueExpr{V: " "},
				},
			}},
			SuppressZero:         false,
			SuppressMissing:      false,
			Mode:                 "DATA_MODE_STRAIGHT",
			InterColumnSortOrder: []int{1, 0}, // ソート順: 1=実績, 0=営業員名
			StateName:            "$",
		},
	}
	hc_hypercube, _ := app.CreateSessionObject(ctx, &hypercube_def)
	hc_layout, _ := hc_hypercube.GetLayout(ctx)
	hc_width := hc_layout.HyperCube.Size.Cx
	hc_height := 1
	if hc_width != 0 {
		hc_height = int(math.Floor(10000 / float64(hc_width)))
	}
	hc_layout.HyperCube.DataPages = []*enigma.NxDataPage{}
	getAllData(ctx, hc_hypercube, hc_layout, hc_width, hc_height, 0)
	renderingHyperCube(hc_layout)
	app.DestroySessionObject(ctx, hc_hypercube.GenericId)

	global.DisconnectFromServer()
}

func getAllList(ctx context.Context, lo_hypercube *enigma.GenericObject, lo_layout *enigma.GenericObjectLayout, w int, h int, lr int) {
	requestPage := []*enigma.NxPage{{Top: lr, Left: 0, Width: w, Height: h}}
	dataPages, _ := lo_hypercube.GetListObjectData(ctx, "/qListObjectDef", requestPage)
	lo_layout.ListObject.DataPages = append(lo_layout.ListObject.DataPages, dataPages[0])
	n := len(dataPages[0].Matrix)
	if lr+n >= lo_layout.ListObject.Size.Cy {
		return
	}
	getAllList(ctx, lo_hypercube, lo_layout, w, h, lr+n)
}

func renderingList(lo_layout *enigma.GenericObjectLayout) {
	hc := lo_layout.ListObject
	allListPages := hc.DataPages
	for _, p := range allListPages {
		for r := range len(p.Matrix) {
			for c := range len(p.Matrix[r]) {
				cell := p.Matrix[r][c]
				field_data := strconv.Itoa(cell.ElemNumber) + ","
				if cell.State == "S" {
					field_data += "(Selected)"
				}
				if cell.ElemNumber == -2 { // -2: the cell is a Null cell.
					field_data += "-"
				} else if !math.IsNaN(float64(cell.Num)) {
					field_data += strconv.FormatFloat(float64(cell.Num), 'f', -1, 64)
				} else if len(cell.Text) > 0 {
					field_data += cell.Text
				} else {
					field_data += ""
				}
				fmt.Println(field_data)
			}
		}
	}
}

func getAllData(ctx context.Context, hc_hypercube *enigma.GenericObject, hc_layout *enigma.GenericObjectLayout, w int, h int, lr int) {
	requestPage := []*enigma.NxPage{{Top: lr, Left: 0, Width: w, Height: h}}
	dataPages, _ := hc_hypercube.GetHyperCubeData(ctx, "/qHyperCubeDef", requestPage)
	hc_layout.HyperCube.DataPages = append(hc_layout.HyperCube.DataPages, dataPages[0])
	n := len(dataPages[0].Matrix)
	if lr+n >= hc_layout.HyperCube.Size.Cy {
		return
	}
	getAllData(ctx, hc_hypercube, hc_layout, w, h, lr+n)
}

func renderingHyperCube(hc_layout *enigma.GenericObjectLayout) {
	hc := hc_layout.HyperCube
	allDataPages := hc.DataPages
	for _, p := range allDataPages {
		for r := range len(p.Matrix) {
			for c := range len(p.Matrix[r]) {
				cell := p.Matrix[r][c]
				field_data := ""
				if cell.ElemNumber == -2 { // -2: the cell is a Null cell.
					field_data += "-"
				} else if len(cell.Text) > 0 {
					field_data += cell.Text
				} else if !math.IsNaN(float64(cell.Num)) {
					field_data += strconv.FormatFloat(float64(cell.Num), 'f', -1, 64)
				} else {
					field_data += ""
				}
				fmt.Println(field_data)
			}
		}
	}
}
