# Typed Catalog launch owner（未發行）

這是隔離的 Compose owner 實作，尚未代表正式 Go 1.27.3 CI、Engine 整合或 runtime 驗收通過。一般 CLI／executor 行為沿原路。

Engine 固定呼叫 `/usr/bin/compose-executor.real --catalog-launch-preview`，只透過 stdin/stdout 傳 server-private JSON，不接 URL、檔案路徑、服務憑證或任意命令。caller 必須限制執行期限、清除繼承憑證／debug 環境、限制 stdout 並丟棄 stderr。

* `phase=inspect`（省略時相同）：`name/environmentName/dockerCompose/rancherCompose/variables`，最多 1.5 MiB。回傳 frozen Compose、顯式 string variables、normal parser 的 config、Docker Inspect、primary/sidekick 關係與建立 skeleton。
* `phase=finalize`：同一來源／變數，另加完整且精確的 `transformedLaunchConfigs` 與 `resolvedSecrets`，最多 10 MiB。Engine 使用原 DockerTransformer／schema projection 提供前者，並以真正 owner／Key/reference 准入提供後者。多餘、缺漏、未知 name 或未解析 reference 均拒絕。
* 所有輸出最多 8 MiB；Compose 原始檔合計最多 1 MiB，插值總量在解析過程限制為 1 MiB，未宣告變數拒絕。錯誤只有固定 code；私有 parser/API body 不回印。

不在 Auth 重寫 YAML 或建立規則：使用既有 `project.Parse/config.Merge/convert.Convert`。Go template 採 bounded、無 host environment／DNS／random／檔案功能的子集，執行錯誤不吞掉；render 後加 `# notemplating` 固定核准內容。context name/version 取原 owner 的 normalization／Catalog config。finalize 與普通 `NormalFactory` 共用 service/volume skeleton、launch config 補值、secret resolver、sidekick、ports 與 service hash。

finalize 的 `serviceRequests[].createBody` 是正常 POST 的精確 DTO；`volumeRequests[].createSkeleton` 沿正常 VolumeTemplate owner。`$createdStack`／`$service:<name>` 只容許 Engine 綁本計畫實際建立的 ID，不能接受模型 ID。`setServiceLinks` 沿同一 parser／DTO，只支援此 graph 內部服務；順序視為集合。正常 owner 會跑兩次 Create 以完成循環 links，因此可先送已存在 targets 的 subset，再送完整 intended links；完成判定仍需完整關聯收斂。

此 candidate 明確拒絕 KVM、provider/driver、host/dependency、build、外部檔案、ignored extends/secret 形狀、外部或跨 Stack／region links、動態 service selectors、未支援的 network graph。這些是尚待 typed adapter 的功能，不能因已有工具就標成完整 Catalog UI 通過。

跨程序授權只使用 Engine 私有 event：`data.catalogLaunchDelegation={planId,stackId,eventId,proof}`。它綁定實際 event/resource，經事件專屬 client 將 proof 放入固定 `X-PastureStack-Catalog-Launch-Delegation` header，含 schema/read/transform/child requests；不修改 router 的共用服務 client。已有 marker 而缺／錯 proof 不退回服務 FULL。delegated context 直接綁已讀回的 exact Stack，避免 name normalization 另建 Stack。

Engine 必須逐 request 驗 nonce/hash、claim plan、Stack/event/frozen graph、原 actor／Key revision／撤權與權限，並在正常 manager 執行前後保留 scope。缺 proof 的拒絕只限 active delegated execution 與真正 launcher caller；真人與一般 Key 後續操作維持原有 RBAC。verified requests 採 metadata-only audit，不能序列化私有 request/response payload。proof 不進模型、公開工具、log 或 WebStorage。

進度與最終 event reply 維持原有 trusted service channel，只送固定文字與該 event 的 reply binding。原 Owner Key 不取得 `publish` 權限；Stack 收到 reply／active 也不是完成證明，Engine 還必須驗 exact graph、ledger、完整 links 與正常 deployment readiness。
