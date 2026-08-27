# 古籍原文语料库（kggraph 参考源）

这些是后续「逐章扫描 → 抽取 → 冲突校验 → 入库」所用的**原文语料**，
不是种子数据。所有入库事实必须带 `source_ref`，指向 `gushu/<书名>/<章节>`。

语料由 kggraph 的前置采集工具 `kggraph source` 管理，主题方向与书单配置在
`corpus.json`，下载结果记录在 `manifest.json`。

## 采集工具：kggraph source

```bash
kggraph source init --topic "周易命理" --dir gushu     # 设定主题方向
kggraph source add --id <book> --title <书名> \
  --provider wikisource --page <页面名>                # 登记书目
kggraph source search --query <关键词>                 # 多来源检索候选
kggraph source fetch --book-id <book> [--force]        # 抓取下载+清洗
kggraph source list                                    # 查看已下载语料
kggraph source verify --book-id <book>                 # 坏段/乱码校验
kggraph source clean                                   # 全量重清洗
```

支持来源：`wikisource`（维基文库全文）、`ctext`（中国哲学书电子化计划，
需在配置中给出 res/chapter 编号）、`yihaiguanyao`（易海观爻，默认剔除白话
译文段落，`--keep-translation` 可保留）。采集逻辑全部由 Go 实现（标准库，
零额外依赖），详见 `internal/source`。

## 已下载书目（30 本，约 310 万字）

| 目录 | 书名 | 版本 / 来源 | 规模 |
| --- | --- | --- | --- |
| `zhouyi` | 周易 | 维基文库，64 卦 + 十翼分章 | 41k |
| `zhouyi-zhengyi` | 周易正义（孔颖达） | 维基文库 | 225k |
| `zhouyi-benyi` | 周易本义（朱熹） | ctext 四库本 | 108k |
| `yixue-qimeng` | 易学启蒙通释 | ctext（含朱熹原文） | 55k |
| `zhouyi-jijie` | 周易集解（四库本） | 维基文库 | 122k |
| `yijing-qiankun-zaodu` | 易纬·乾坤凿度（四库本） | 维基文库 | 6k |
| `wuxing-dayi` | 五行大义（萧吉） | 维基文库，五卷 | 81k |
| `yuanhai-zipping` | 渊海子平 | 维基文库 | 61k |
| `sanming-tonghui` | 三命通会（十二卷） | 卷一至九维基文库 + 卷十至十二 ctext 四库本拼接 | 315k |
| `ditian-sui` | 滴天髓（辑要） | 维基文库 | 14k |
| `ditian-sui-chanwei` | 滴天髓阐微（任铁樵全本） | 维基文库 | 88k |
| `ziping-zhenquan` | 子平真诠评注 | ctext（徐乐吾注本） | 28k |
| `qiongtong-baojian` | 穷通宝鉴 | 维基文库 | 36k |
| `shenfeng-tongkao` | 神峰通考 | 维基文库 | 157k |
| `xingping-huihai` | 增补星平会海命学全书 | ctext | 229k |
| `ziwei-doushu` | 紫微斗数全书 | 维基文库，三卷 | 84k |
| `xingxue-dacheng` | 星学大成（四库本） | 维基文库 | 249k |
| `zhangguo-xingzong` | 张果星宗 | 维基文库 | 196k |
| `meihua-yishu` | 梅花易数 | 维基文库，三卷 | 44k |
| `zengshan-buyi` | 增删卜易 | 维基文库 | 137k |
| `bushi-zhengzong` | 卜筮正宗 | 易海观爻全文（剔除白话译文，凡例/目录由 ctext 卷一补充） | 64k |
| `huozhulin` | 火珠林 | 维基文库 | 22k |
| `liuren-zhinan` | 六壬指南 | 维基文库 | 25k |
| `liuren-daquan` | 六壬大全 | 维基文库 | 294k |
| `yanbo-diaosou-ge` | 烟波钓叟歌（奇门） | 维基文库 | 2k |
| `jiaoshi-yilin` | 焦氏易林 | ctext（四卷全） | 88k |
| `huangji-jingshi` | 皇极经世 | 维基文库 | 269k |
| `jingshi-yizhuan` | 京氏易传 | 维基文库 | 19k |
| `xieji-bianfangshu` | 钦定协纪辨方书（四库全书本） | 维基文库，三十六卷 | 230k |
| `mayi-xiangfa` | 麻衣相法 | 易海观爻（23 小节） | 88k |

每本书目录下均有分章节 `*.txt` 与合并全文 `full.txt`；统计见 `manifest.json`。

## 质量与使用注意

1. 除 `qiongtong-baojian`、`xieji-bianfangshu` 外，正文均为繁体/异体原文；提取入库时按需转简体。
2. `bushi-zhengzong` 已换用易海观爻干净全文（原 ctext 四卷版卷二、卷三存在转录乱码，已弃用）；章节按原书小节划分，后半部「黄金策直解/十八问答/卦例」并入末章，粒度较粗但内容完整。
3. `ditian-sui` 为《滴天髓辑要》本；全注本见 `ditian-sui-chanwei`（任铁樵《阐微》）。
   `ziping-zhenquan` 为评注本，非纯原文。
4. `sanming-tonghui` 卷十至十二来自 ctext 四库本，与卷一至九排版、异体字略有差异，不影响抽取。
5. 经典机制性规则（五行生克、排盘、冲合）按高置信抽取；经验断语（神煞吉凶、体用吉凶等）标低置信并注明争议来源。
