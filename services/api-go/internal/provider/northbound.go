package provider

import (
	"fmt"
	"time"
)

// NorthboundFlow 沪深港通北向资金摘要。
type NorthboundFlow struct {
	NetInflow   float64 `json:"netInflow"`   // 亿元
	SHNetInflow float64 `json:"shNetInflow"` // 沪股通 亿元
	SZNetInflow float64 `json:"szNetInflow"` // 深股通 亿元
	AsOf        string  `json:"asOf"`
	Status      string  `json:"status"` // inflow / outflow / flat
	Text        string  `json:"text"`
	Source      string  `json:"source"`
}

func (b *Bundle) NorthboundFlow() (*NorthboundFlow, error) {
	// 东财实时北向资金（单位：元，这里换算成亿元）
	url := "https://push2.eastmoney.com/api/qt/kamt.rtmin/get?fields1=f1,f2,f3,f4&fields2=f51,f52,f53,f54,f55,f56&ut=b2884a393a59ad64002292a3e90d46a5"
	var payload map[string]interface{}
	if err := getJSON(b.Client, url, "https://data.eastmoney.com/", &payload); err != nil {
		return fallbackNorthboundHistory(b)
	}
	data, _ := payload["data"].(map[string]interface{})
	if data == nil {
		return fallbackNorthboundHistory(b)
	}
	s2n := asFloat(data["s2n"]) // 北向净流入（元）部分接口字段
	// 优先用当日累计：hk2sh / hk2sz 若存在
	hk2sh := asFloat(data["hk2sh"])
	hk2sz := asFloat(data["hk2sz"])
	if hk2sh == 0 && hk2sz == 0 {
		// rtmin 序列最后一笔
		raw, _ := data["s2n"].([]interface{})
		if len(raw) == 0 {
			raw, _ = data["klines"].([]interface{})
		}
		_ = raw
	}
	// 另一路：bas 接口更稳
	if flow, err := b.northboundBas(); err == nil && flow != nil {
		return flow, nil
	}
	if s2n == 0 && hk2sh == 0 && hk2sz == 0 {
		return fallbackNorthboundHistory(b)
	}
	net := (hk2sh + hk2sz) / 1e8
	if net == 0 {
		net = s2n / 1e8
	}
	return makeNorthbound(net, hk2sh/1e8, hk2sz/1e8, time.Now().Format("15:04")), nil
}

func (b *Bundle) northboundBas() (*NorthboundFlow, error) {
	url := "https://push2.eastmoney.com/api/qt/kamtbas.rtmin/get?fields1=f1,f2,f3,f4&fields2=f51,f52,f53,f54,f55,f56&ut=b2884a393a59ad64002292a3e90d46a5"
	var payload map[string]interface{}
	if err := getJSON(b.Client, url, "https://data.eastmoney.com/", &payload); err != nil {
		return nil, err
	}
	data, _ := payload["data"].(map[string]interface{})
	if data == nil {
		return nil, fmt.Errorf("northbound bas empty")
	}
	// data.s2n 可能是数组 "时间,沪,深,北向"
	raw, _ := data["s2n"].([]interface{})
	if len(raw) == 0 {
		return nil, fmt.Errorf("northbound bas s2n empty")
	}
	last := asString(raw[len(raw)-1])
	parts := splitCSV(last)
	if len(parts) < 4 {
		return nil, fmt.Errorf("northbound bas parse fail")
	}
	sh := parseF(parts[1]) / 1e8
	sz := parseF(parts[2]) / 1e8
	net := parseF(parts[3]) / 1e8
	asOf := parts[0]
	return makeNorthbound(net, sh, sz, asOf), nil
}

func fallbackNorthboundHistory(b *Bundle) (*NorthboundFlow, error) {
	url := "https://datacenter-web.eastmoney.com/api/data/v1/get?sortColumns=TRADE_DATE&sortTypes=-1&pageSize=1&pageNumber=1&reportName=RPT_MUTUAL_DEAL_HISTORY&columns=ALL&source=WEB&client=WEB"
	var payload map[string]interface{}
	if err := getJSON(b.Client, url, "https://data.eastmoney.com/", &payload); err != nil {
		return nil, err
	}
	result, _ := payload["result"].(map[string]interface{})
	rows, _ := result["data"].([]interface{})
	if len(rows) == 0 {
		return nil, fmt.Errorf("northbound history empty")
	}
	m, _ := rows[0].(map[string]interface{})
	// 字段单位多为万元或亿元，东财历史接口 NET_DEAL_AMT 多为亿元
	net := asFloat(m["NET_DEAL_AMT"])
	sh := asFloat(m["CCC_NET_DEAL_AMT"])
	sz := asFloat(m["SZ_NET_DEAL_AMT"])
	asOf := asString(m["TRADE_DATE"])
	if len(asOf) > 10 {
		asOf = asOf[:10]
	}
	return makeNorthbound(net, sh, sz, asOf), nil
}

func makeNorthbound(net, sh, sz float64, asOf string) *NorthboundFlow {
	status := "flat"
	text := "北向资金接近平衡"
	if net > 5 {
		status = "inflow"
		text = fmt.Sprintf("北向资金净流入约 %.1f 亿元，偏积极", net)
	} else if net > 0 {
		status = "inflow"
		text = fmt.Sprintf("北向资金小幅净流入约 %.1f 亿元", net)
	} else if net < -5 {
		status = "outflow"
		text = fmt.Sprintf("北向资金净流出约 %.1f 亿元，偏谨慎", -net)
	} else if net < 0 {
		status = "outflow"
		text = fmt.Sprintf("北向资金小幅净流出约 %.1f 亿元", -net)
	}
	return &NorthboundFlow{
		NetInflow:   net,
		SHNetInflow: sh,
		SZNetInflow: sz,
		AsOf:        asOf,
		Status:      status,
		Text:        text,
		Source:      "eastmoney",
	}
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(s[i])
	}
	out = append(out, cur)
	return out
}
