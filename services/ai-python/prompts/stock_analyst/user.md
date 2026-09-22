股票：{{stock.name}} {{stock.symbol}}，行业 {{stock.industry}}，现价 {{stock.price}}，涨跌 {{stock.changePercent}}%，换手 {{stock.turnover}}%
指标：MA5/10/20={{indicators.ma5}}/{{indicators.ma10}}/{{indicators.ma20}}，RSI6={{indicators.rsi6}}，乖离={{indicators.bias5}}，MACD柱={{indicators.hist}}，多头排列={{indicators.bullAlign}}
算法：
{{algorithm_lines}}
板块与涨跌归因上下文：
{{sector_lines}}
请按系统要求返回完整中文 JSON：必须包含板块分析与涨跌归因（板块/题材/公告/个股），禁止只返回 {}，禁止编造未给出的公告或涨跌幅。
