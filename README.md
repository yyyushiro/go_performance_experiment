# はじめに
これは「デートがマンネリ化したカップルにランダムで一つデートプランを提示する」アプリのためのバックエンドAPIである。

ランダムである以上ユーザーごとに何か対応する必要がないため、叩けるAPIはランダムにデートプランが出てくる `/datePlan/` のみである。

そして僕のアイデアがたまるまでランダム化するほど件数が稼げないため、この単純さを活かして「SQL文によるDBの性能改善」をテーマとしたポートフォリオに転用することにした。

# 環境
- OS: macOS
- Language: GO version go1.25.6 darwin/arm64
- Database: SQLite3 (modernc.org/sqlite)
- editor: VScode + Go Extension
- Loat Test: hey

## 2/16
MVP(`/datePlan/` のみ)の実装。First commit.

## 2/17
```
Summary:
  Total:        436.5398 secs
  Slowest:      12.5233 secs
  Fastest:      8.5823 secs
  Average:      11.7876 secs
  Requests/sec: 4.2310
  
  Total data:   379574 bytes
  Size/request: 205 bytes

Response time histogram:
  8.582 [1]     |
  8.976 [1]     |
  9.370 [1]     |
  9.765 [3]     |
  10.159 [4]    |
  10.553 [4]    |
  10.947 [13]   |■
  11.341 [22]   |■
  11.735 [733]  |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  12.129 [896]  |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  12.523 [169]  |■■■■■■■■


Latency distribution:
  10%% in 11.6198 secs
  25%% in 11.6806 secs
  50%% in 11.7621 secs
  75%% in 11.9052 secs
  90%% in 12.1158 secs
  95%% in 12.1959 secs
  99%% in 12.3350 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0001 secs, 0.0000 secs, 0.0055 secs
  DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0019 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0005 secs
  resp wait:    11.7874 secs, 8.5823 secs, 12.5232 secs
  resp read:    0.0000 secs, 0.0000 secs, 0.0021 secs

Status code distribution:
  [200] 1847 responses
  ```
以上は合計アクセス数1847件、同時接続数50人の場合の負荷テストの結果である。
`/datePlan` APIのみを叩き続けるものであり、DBには100,000件のデータが入っている。

Requests/secで表されるスループットはおよそ４件。
また、Detailsのresp waitで表されるGoプログラムとDBの合計処理時間の合計はおよそ12秒であることがわかった。

単純にデータを一件返すだけのAPIであることを考えるとこの速度には改善の余地があると考えられる。

ここで、現在のSQL文に注目する。

`SELECT id, title, content FROM datePlans ORDER BY RANDOM() LIMIT 1`

ボトルネックになっているのは `ORDER BY　RANDOM()` の部分であると考えられる。

この部分で何が起こっているのか述べる。まず、DBは対象のデータ群に対して[ランダムな数値｜対象データ]という形で一つ一つにペアを作り、それをメモリ上に作ったB-Treeに放り込んでソートする（対象データが重い場合はランダムな数値のみがメモリに来る）。そして、一番値が小さい一件のみを抽出し、それを返す。

データ数を $N$ とすれば、これは $O(NlogN)$ の計算量がかかる。また、メモリ使用量も $O(N)$である。実際にはメモリにデータが乗り切らずストレージとスワップしながらソートする可能性もあり、さらなるパフォーマンスもありうる。

改めて、ただ一件のみを抽出するためだけにこの計算量とメモリ使用量を消費するのは非効率である。

ただしこの速度をB-treeインデックスで解決することはできない。なぜならば並べ替えはランダムであり、既存のB-treeはランダムソートをスキップすることができないからだ。

したがってデータ自体を並べ替えるのではなく、`id` を一件ランダムに指定して一件のみDBから抽出するという方法が適切であると考える。

SQL文は以下のようになる。

`SELECT COUNT(id) FROM datePlans`

`SELECT id, title, content FROM datePlans WHERE id = ?`

一文目は現在の行数をチェックするもの。その行数を利用して、ランダムに対象となるIDを生成する。

二文目はそのIDに該当するデータを得るもの。

これらを反映して、同様に合計アクセス数2000件、同時接続数50人という条件で負荷テストを行った結果が以下である。

```
Summary:
  Total:        97.6657 secs
  Slowest:      5.2664 secs
  Fastest:      0.1082 secs
  Average:      2.4029 secs
  Requests/sec: 20.4780
  
  Total data:   411071 bytes
  Size/request: 205 bytes

Response time histogram:
  0.108 [1]     |
  0.624 [9]     |■
  1.140 [15]    |■
  1.656 [32]    |■■
  2.171 [702]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  2.687 [630]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.203 [524]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.719 [43]    |■■
  4.235 [19]    |■
  4.751 [14]    |■
  5.266 [11]    |■


Latency distribution:
  10%% in 1.8462 secs
  25%% in 1.9730 secs
  50%% in 2.3692 secs
  75%% in 2.7806 secs
  90%% in 2.9358 secs
  95%% in 3.1248 secs
  99%% in 4.3707 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0001 secs, 0.0000 secs, 0.0066 secs
  DNS-lookup:   0.0001 secs, 0.0000 secs, 0.0026 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0004 secs
  resp wait:    2.4028 secs, 0.1082 secs, 5.2663 secs
  resp read:    0.0000 secs, 0.0000 secs, 0.0024 secs

Status code distribution:
  [200] 2000 responses
```

スループットは20件となり5倍のパフォーマンス。

平均処理時間は2.4秒となりこれも5倍のパフォーマンスである。

これは大きな改善であると言える。

これらSQL文の計算量は、一文目に対して $O(N)$, 二文目に対して $O(1)$ であり、パフォーマンスの伸びに対応していると言える。


デートプランを取得するたびに行数を計算するのは非効率なため、それらの操作を切り離す必要がある。
今回は行数をglobal variableとして保持することでそれを解決した。

```
Summary:
  Total:        0.0723 secs
  Slowest:      0.0106 secs
  Fastest:      0.0000 secs
  Average:      0.0017 secs
  Requests/sec: 27644.4666
  
  Total data:   411410 bytes
  Size/request: 205 bytes

Response time histogram:
  0.000 [1]     |
  0.001 [930]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.002 [482]   |■■■■■■■■■■■■■■■■■■■■■
  0.003 [272]   |■■■■■■■■■■■■
  0.004 [167]   |■■■■■■■
  0.005 [86]    |■■■■
  0.006 [35]    |■■
  0.007 [10]    |
  0.008 [14]    |■
  0.010 [1]     |
  0.011 [2]     |


Latency distribution:
  10%% in 0.0001 secs
  25%% in 0.0005 secs
  50%% in 0.0012 secs
  75%% in 0.0025 secs
  90%% in 0.0038 secs
  95%% in 0.0048 secs
  99%% in 0.0071 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0023 secs
  DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0014 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0004 secs
  resp wait:    0.0016 secs, 0.0000 secs, 0.0085 secs
  resp read:    0.0000 secs, 0.0000 secs, 0.0005 secs

Status code distribution:
  [200] 2000 responses
```

行数の計算が一回きりになったため、パフォーマンスが大幅に向上した。
