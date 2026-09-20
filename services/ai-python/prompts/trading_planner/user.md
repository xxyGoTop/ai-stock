请为下面这只股票起草一份模拟交易计划，价格必须贴近现价，不要脱离给定数据。
股票：{{stock.name}} {{stock.symbol}}，现价 {{stock.price}}，涨跌 {{stock.changePercent}}%
指标：MA5/10/20={{indicators.ma5}}/{{indicators.ma10}}/{{indicators.ma20}}，RSI6={{indicators.rsi6}}，乖离={{indicators.bias5}}
算法：
{{algorithm_lines}}
只返回 JSON。
