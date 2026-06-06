# tddmaster Refactor Planı — pentestmaster mimarisine taşıma

## Context

`tddmaster` (AI ile geliştirilmiş) ile `pentestmaster` (kullanıcının kendi geliştirdiği) aynı kök dizinde yan yana duruyor. tddmaster çalışıyor ama mimarisi bozuk:

- **State dağınık ve mükerrer**: `.tddmaster/.state/state.json` (global), `.tddmaster/specs/<spec>/progress.json` (per-spec), `manifest.yml` arasında aynı veri (task completion, decisions, spec adı/açıklaması, phase) birden çok yerde tutuluyor.
- **State machine if-else cehennemi**: `cmd/next.go` içinde `handleAnswer` → phase başına `handleXxxAnswer` fonksiyonları, 200+ satır iç içe if-else. "Important gate" bir phase değil, answer handler'ların içine sokulmuş bir interception. TDD red/green/refactor StateFile phase'i değil, task-level runtime context. Yeni feature eklemek imkânsız.
- **manifest.yaml hiyerarşisi yanlış**: `skipVerify`, `maxRefactorRounds`, `maxVerificationRetries` global olması gerekirken `tddmaster.tdd.*` altına 4 seviye gömülü. Format YAML.
- **Promptlar dağınık**: `internal/sync/adapters/shared/*.go`, `internal/context/service/tdd/instructions.go`, `internal/context/model/strings.go`, `internal/context/service/meta/roadmap.go` — Go string fonksiyonlarına serpiştirilmiş, tek merkez yok.
- **Agent isimleri tutarsız**: çoğu `tddmaster-*` ama test yazarı bare `test-writer`.

**Hedef**: pentestmaster'ın **phase → module → step** mimarisini, tek-kaynak JSON state'ini, merkezi prompt registry'sini ve manifest.json toggle sistemini tddmaster'a birebir uygulayarak **kök dizine** (`/Users/pragmata/Projeler/tddmaster-refactor/`) refactor edilmiş tddmaster'ı yazmak. İki dizinden de kopya çekilecek; mantık tddmaster'dan, iskelet/mimari pentestmaster'dan.

### Onaylanmış kararlar (kullanıcı)

1. **TDD döngü modeli**: **Hibrit** — discovery/refinement/spec fazları lineer step; sadece `executing` fazı içinde özel task alt-döngüsü (task-iterator cursor).
2. **Go module path**: mevcut korunur → `module github.com/pragmataW/tddmaster`.
3. **Kapsam**: tüm akış baştan yazım — discovery → refinement → spec-proposal → spec-approved → executing(TDD+gate) → blocked → completed.

---

## Hedef Mimari (pentestmaster'dan birebir)

### Motor (engine) — pentestmaster `internal/engine/` kopyası

pentestmaster motorunun çekirdek tipleri (`internal/engine/phase.go|module.go|step.go|context.go|action.go`) birebir alınır:

```go
type PhaseDef  struct { ID PhaseID; Modules []ModuleDef }
type ModuleDef struct { ID ModuleID; Steps []StepDef }
type StepDef   struct {
  ID       StepID
  Prompt   func(c *Context) Action
  Validate func(answer []byte) error
  Emit     func(answer []byte) (json.RawMessage, error)
  Run      func(c *Context, decision []byte) (RunResult, error)
}
```

`Context.Next()` / `Context.Submit()` döngüsü (pentestmaster `context.go:221-340`) aynen alınır:
- İlk cevaplanmamış step'i bul.
- `Run != nil` → otomatik çalıştır, `Done` ise persist+advance.
- Aksi halde `Prompt(c)` action'ı kullanıcıya dön.
- `Submit()` → `Validate` → `Emit` → `persistStepAndAdvance` → `Next()`.

**Anlık kayıt**: `persistStepAndAdvance` (pentestmaster `context.go:160-206`) her step'ten sonra `SaveProgress` + (faz tamamlanınca) `SaveTarget` yapar. tddmaster'da bu korunur → **her step anında diske yazılır**. Step cevabı (verifier raporu, plan, vb. dahil) doğrudan ilgili progress dosyasındaki `StepProgress.Answer` alanına yazılır; ayrı dosya yok.

### Hibrit: executing fazı — GENERIC iterator, TDD'ye özel hardcode YOK

> **Mimari ilke (kullanıcının vurgusu): yeni bir red/green/refactor benzeri akış eklemek, çıkarmak, kapatmak "tak-çıkar" olmalı. Motor `red`/`green`/`refactor`/`gate` diye bir şey BİLMEMELİ.** Bunlar yalnızca katalog verisi (StepDef) olmalı; mantık deklaratif.

executing fazı için motora eklenen TEK genişletme **generic bir iterator** — TDD'ye özel değil. `ModuleDef`'e (veya bir module-grubuna) opsiyonel alan:

```go
type ModuleDef struct {
  ID       ModuleID
  Steps    []StepDef
  Iterator IteratorID   // "" = normal lineer modül; "tasks" = collection üzerinde tekrarla
}
```

- `Iterator != ""` olan bir modül-grubu, motor tarafından **cursor'daki her item için** step'lerini tekrar çalıştırır. Item kaynağı (`tasks`) bir `IteratorProvider` ile çözülür — generic kayıt: `engine.RegisterIterator("tasks", func(c)*[]Item)`. Motor item'ın ne olduğunu bilmez.
- `red`, `green`, `refactor`, `importantGate` → hepsi bu iterating grubun **sıradan StepDef'leri**. Her biri `DelegateAgent` + `InstructionKey` taşır. Motor bunların TDD olduğunu bilmez; sadece "iterator grubunun step'leri" görür.
- Cursor + per-item progress `executing.json` içinde: `{phase, cursor:{index, item}, modules[...]}`. Bir item'ın tüm enabled step'leri `answered` olunca cursor ilerler, stepler reset.

#### do-while yapısı: setup → body (iterated) → teardown

> **Kullanıcı vurgusu: döngü do-while gibi — başlangıçta bir kez yapılacaklar, döngüye dahil olanlar, döngü sonunda bir kez yapılacaklar.** Bu, motora yeni kavram eklemeden **sadece modül sıralamasıyla** çıkar: motor bir fazın modüllerini sırayla işler; iterator'sız modül bir kez, iterator'lı modül her item için çalışır.

executing fazı modül sırası:

```
phase: executing
  module: setup      (Iterator="")        → BİR KEZ (döngü başı: branch hazırlığı, task listesi kilitleme, ön-koşul)
  module: cycle      (Iterator="tasks")   → HER TASK İÇİN (do-while gövdesi)
       steps: importantGate → red → green → refactor
  module: finalize   (Iterator="")        → BİR KEZ (döngü sonu: bütünleşik test, özet, completed'a geçiş hazırlığı)
```

- **setup** (`başlangıçta yapılacaklar`): iterator'sız, fazın ilk modülü → motorun lineer akışı gereği tüm task'lardan önce bir kez cevaplanır.
- **cycle** (`döngüye dahil olanlar`): iterator'lı gövde. Her task için step'leri tekrar. do-while semantiği: koşul = "cursor'da işlenmemiş item var mı". Sıfır task → gövde hiç çalışmaz, setup/finalize yine çalışır.
- **finalize** (`döngü sonunda yapılacaklar`): iterator'sız, fazın son modülü → tüm task'lar bitince bir kez.
- Aynı desen **her iterating faza** uygulanır (örn. ileride bir `review` fazı: setup→per-item review→finalize). setup/body/teardown'a step eklemek/çıkarmak yine sadece katalog+manifest.

#### Chain of Responsibility — her şey `next` ile bağlı

> **Kullanıcı vurgusu: setup, cycle, finalize birbirine bağlı bir zincir; ayrıca her birinin İÇİNDEKİ step'ler de kendi aralarında bağlı.** Yani iki seviyeli Chain of Responsibility. Ekleme/çıkarma = zinciri yeniden bağlama (splice), motor kodu değişmez.

Katalog, dizi yerine **bağlı zincir (linked handler chain)** olarak yorumlanır. Her handler kendi işini görür veya zinciri ilerletir:

```go
type Link struct {
  Node    any        // StepDef | ModuleDef
  Next    NodeID     // bir sonraki halka ("" = zincir sonu → üst seviyeye dön)
  Enabled bool       // manifest/spec override; disabled link otomatik bypass (zincirde atlanır)
}
```

İki seviye:
- **Dış zincir (faz içi modüller)**: `setup → cycle → finalize → (faz sonu)`. cycle halkası özel: cursor'da işlenmemiş item varken **kendine geri bağlanır** (self-loop = döngü), bitince `Next`'e (finalize) geçer.
- **İç zincir (modül içi step'ler)**: `importantGate → red → green → refactor → (modül sonu)`. Her step bir sonrakine bağlı.

**Çözümleme**: `Context.Next()` aktif zincir konumundan başlar, ilk `Enabled && !Answered` handler'a yürür. Disabled handler `Next`'e şeffaf bypass — bu yüzden `refactor`'ü kapatmak zinciri kırmaz, `green → finalize`'a otomatik akar. pentestmaster'ın `Next()/Submit()` döngüsü (array tarama) bu zincir-yürümesinin somut implementasyonu; `Next` alanı default olarak katalog sırasındaki bir sonraki düğüm, gerektiğinde explicit override edilebilir.

**Tak-çıkar = splice**: yeni step `lint`'i `green` ile `refactor` arasına sokmak → `green.Next = lint`, `lint.Next = refactor`. Tek katalog düzenlemesi; ne motor ne komşu step impl'leri değişir. Çıkarmak → komşuları yeniden bağla (veya sadece `Enabled=false` ile bypass).

**Sonuç — tak-çıkar matrisi (motor değişmez, kod yazılmaz):**

| İşlem | Yapılacak tek şey |
|-------|-------------------|
| Yeni cycle step ekle (ör. `green`↔`refactor` arası `lint`) | `catalog.go`'ya 1 StepDef + 1 InstructionKey + manifest'te toggle. Motor/CLI'ye dokunma. |
| `refactor` çıkar/kapat | manifest (veya spec override) `phases.executing.modules.refactor.enabled=false`. Kod yok. |
| `importantGate` ekle/çıkar | Zaten bir step; toggle. Kod yok. |
| Komple yeni iterating akış (ör. "security-pass") | Yeni ModuleDef `{Iterator:"tasks"}` + stepler + instructionlar. Motor değişmez. |
| Yeni faz ekle (ör. `review`) | `ids.go` + `catalog.go` + `stepImpls` girişi + manifest. pentestmaster'daki gibi saf veri. |

> Diğer tüm fazlar (discovery/refinement/spec/blocked/completed) saf lineer — pentestmaster ile birebir. Sadece executing iterator kullanır; o da TDD'ye değil generic "tasks" collection'ına bağlı.

### State — tek kaynak (pentestmaster `internal/state/state.go` deseni)

Mükerrer state YOK. Tek hiyerarşi:

```
.tddmaster/
  state.json                      # global: activeSpec, specs set (pentestmaster state.json eşi)
  manifest.json                   # phase/module/step toggle + global config (JSON!)
  {spec-slug}/
    state.json                    # spec state: phase, slug, createdAt, completionReason
    progress/
      discovery.json
      discoveryRefinement.json
      specProposal.json
      specApproved.json
      executing.json              # task cursor + per-task red/green/refactor progress
      blocked.json
      completed.json
    spec.md                       # türev render (kaynak değil)
```

- `state.json` (global): `{version, activeSpec, specs:{}}`.
- `{slug}/state.json`: `{slug, phase, createdAt, completionReason?}`.
- Progress dosyaları: pentestmaster `PhaseProgress{phase, modules[]{module, steps[]{step, answered, answer}}}` şeması birebir.
- **Tek yazım noktası**: `internal/state/` paketi `Load/Save/SaveProgress/SaveTarget` fonksiyonları. Başka hiçbir yerden state dosyası yazılmaz.
- **Silinen mükerrerlikler**: `state.json`'daki `completedTasks[]`, `decisions[]`, `specDescription`, `transitionHistory` artık progress dosyalarından türetilir; ayrı tutulmaz.

### Manifest — JSON + düz hiyerarşi (pentestmaster `internal/manifest/manifest.go` deseni)

YAML → JSON. `tdd.*` altındaki global ayarlar **tepe seviyeye** çıkar. Phase/module/step toggle pentestmaster `PhaseConfig/ModuleConfig` şemasıyla:

```json
{
  "version": 1,
  "command": "tddmaster",
  "tools": ["claude-code", "antigravity"],
  "allowGit": false,
  "skipVerify": false,
  "maxVerificationRetries": 0,
  "maxRefactorRounds": null,
  "maxIterationsBeforeRestart": 15,
  "tddMode": true,
  "importantTaskGate": true,
  "verifyCommand": null,
  "injectProjectConventions": true,
  "project": { "languages": ["go"], "frameworks": [], "ci": [], "testRunner": null },
  "concerns": ["open-source", "beautiful-product"],
  "phases": {
    "discovery":        { "enabled": true,  "modules": { "listen": {"enabled": true, "steps": {"context": true}}, "questions": {"enabled": true, "steps": {...}} } },
    "discoveryRefinement": { "enabled": true, "modules": { "premises": {...}, "alternatives": {...} } },
    "specProposal":     { "enabled": true,  "modules": {...} },
    "specApproved":     { "enabled": true,  "modules": { "importantBulk": {...} } },
    "executing":        { "enabled": true,  "modules": { "importantGate": {"enabled": true}, "red": {"enabled": true}, "green": {"enabled": true}, "refactor": {"enabled": true} } },
    "blocked":          { "enabled": true,  "modules": {...} },
    "completed":        { "enabled": true,  "modules": {...} }
  }
}
```

- `skipVerify` artık tepe seviyede global flag → tek yerden toggle.
- **Spec bazında kapatma**: manifest global toggle'a ek olarak, her spec için override. `{slug}/state.json` (veya `{slug}/overrides.json`) içinde `disabledSteps: ["executing.refactor", ...]`. `phases.Enabled(manifest, specOverrides)` global + spec override'ı birleştirip filtreler. **Important gate, TDD refactor, skipVerify — hepsi step ve spec bazında kapatılabilir.**

### Phase/Module/Step kataloğu (tddmaster akışı)

pentestmaster `internal/phasecatalog/{ids.go,catalog.go,config.go}` deseninde, tddmaster fazlarıyla:

| Phase | Modules → Steps | Step tipi |
|-------|-----------------|-----------|
| `discovery` | `listen` → [`context`]; `questions` → [her `Questions[]` ID'si ayrı step: `status_quo`, `ambition`, `reversibility`, `user_impact`, `verification`, `scope_boundary`, `edge_cases` + concern extras] | question (Prompt/Validate/Emit) |
| `discoveryRefinement` | `premises` → [her premise ayrı step]; `alternatives` → [`select`] | question |
| `specProposal` | `draft` → [`approve`] | question |
| `specApproved` | `importantBulk` → [`mark`] (gate açıksa) | question |
| `executing` | `setup`(Iterator="") → [bir-kez ön adımlar]; **`cycle`(Iterator="tasks")** → `importantGate`[`plan`] → `red`[`write-tests`] → `green`[`implement`,`verify`] → `refactor`[`apply`,`recheck`]; `finalize`(Iterator="") → [bir-kez kapanış] | delegate (agent spawn) |
| `blocked` | `resolve` → [`decision`] | question |
| `completed` | `finalize` → [`summary`] | auto-run |

- **Discovery soruları step oldu**: her soru `Questions[]`'tan (`internal/context/model/questions.go`) bir StepDef. Cevap anında `discovery.json`'a yazılır.
- **Important gate step oldu**: `executing.importantGate.plan` — artık interception değil, gerçek step. `delegateAgent: tddmaster-planner`. Gate kapalıysa manifest/spec override ile step disabled.
- **TDD red/green/refactor step oldu**: `executing` altında modüller. `skipVerify` true ise `green.verify` / `refactor.recheck` stepleri spec/global override ile atlanır.

### Prompt Registry — tek merkez (pentestmaster `internal/promptregistry/` deseni)

Tüm promptlar **tek pakette** toplanır: `internal/promptregistry/`. pentestmaster gibi `instructions_*.go` + `instruction_registry.go` + `keys.go`.

**Promptlar tddmaster'dan BİREBİR ve EKSİKSİZ alınır.** Kaynak → hedef eşlemesi:

| Kaynak (tddmaster) | İçerik |
|--------------------|--------|
| `internal/sync/adapters/shared/executor_prompt.go` `ExecutorInstructions` | executor body |
| `internal/sync/adapters/shared/test_writer.go` `TestWriterInstructions` | test-writer body |
| `internal/sync/adapters/shared/verifier_prompt.go` `VerifierInstructions*`, `Verifier{Red,Green,Refactor}PhaseBlock` | verifier bodies |
| `internal/sync/adapters/shared/planner_prompt.go` `PlannerInstructions` | planner body |
| `internal/sync/adapters/shared/agents_md.go` `BuildProtocol/Coaching/Rules/Section` | protocol/coaching |
| `internal/context/service/tdd/instructions.go` `Verifier{Red,Green,Refactor}PhaseInstruction`, `VerifierReportSchemaJSON/Rules` | TDD phase instructions + verifier rapor şeması |
| `internal/context/model/strings.go` (TDD delegation table, gate kuralları, `TDDPhaseGreenInstruction` vb.) | behavioral kurallar |
| `internal/context/service/meta/roadmap.go` `BuildRoadmap` | roadmap metni |
| `internal/context/service/meta/{protocol,gate,interactive}.go` | meta promptlar |
| `internal/context/service/discovery/*.go` (questions, enrichments, review, prefill) | discovery promptları |

→ Hepsi `promptregistry.Instructions[key]` ve `promptregistry.AgentSpec.Body` olarak taşınır. **Metin değiştirilmez, sadece yer değiştirir.**

### Agent Registry & isimlendirme (pentestmaster `agent_registry.go` deseni)

pentestmaster `pentestmaster-{domain}-{function}` deseni. tddmaster'da **tüm agentler `tddmaster-*`**:

```go
const (
  AgentTestWriter AgentRegistryKey = "tddmaster-test-writer"   // bare "test-writer" DÜZELTİLDİ
  AgentExecutor   AgentRegistryKey = "tddmaster-executor"
  AgentVerifier   AgentRegistryKey = "tddmaster-verifier"
  AgentPlanner    AgentRegistryKey = "tddmaster-planner"
)
```

`AgentSpec{Description, Tools, Model, Body}` pentestmaster ile birebir. Step'ler `Action{DelegateAgent: "tddmaster-..."}` döner. **Bare `test-writer` referansları** (`internal/context/model/strings.go:83,84,100,166` ve sync adapter'ları) `tddmaster-test-writer`'a güncellenir.

---

## Yapılacaklar — Task Listesi

### Faz 0 — İskelet kurulum (kök dizin)
- **T0.1** Kök dizinde `go.mod` oluştur: `module github.com/pragmataW/tddmaster` (mevcut path korunur), tddmaster'ın `go.sum` bağımlılıkları kopyalanır.
- **T0.2** `main.go` + `cmd/root.go` (cobra) — pentestmaster `cmd/` iskeleti referans, tddmaster komut isimleri (`spec`, `next`, `init`, `sync`, `config`, `undo` vb.).
- **T0.3** Dizin iskeleti: `internal/{engine,engine/phases,phasecatalog,state,manifest,promptregistry,paths,scaffold,context,spec,sync,output}`.

### Faz 1 — Motor (pentestmaster engine birebir)
- **T1.1** `internal/engine/{phase,module,step,action,context}.go` pentestmaster'dan kopyala; import path `github.com/pragmataW/tddmaster/...` yap.
- **T1.2** `internal/engine/context.go` `Next/Submit/persistStepAndAdvance/bootstrapPhaseProgress/alignProgress` aynen al.
- **T1.3** **Chain of Responsibility**: `Link{Node, Next, Enabled}` + iki seviyeli zincir yürüme. Disabled handler şeffaf bypass. `Next` default = katalog sırası, explicit override edilebilir. (pentestmaster array taraması bunun tabanı.)
- **T1.4** **Generic iterator**: `engine.RegisterIterator("tasks", ...)`, `ModuleDef.Iterator`, cycle halkası self-loop (do-while), cursor exhausted → `Next`. Motor TDD/red/green/refactor string'i içermez.
- **T1.5** Engine unit testleri (pentestmaster `engine_test.go`, `autoexec_test.go` uyarlanır) + zincir splice/bypass + iterator do-while testleri.

### Faz 2 — State (tek kaynak)
- **T2.1** `internal/state/state.go`: `State`(global), `SpecState`(per-spec), `PhaseProgress/ModuleProgress/StepProgress`. `Load/Save/LoadTarget/SaveTarget/LoadProgress/SaveProgress`.
- **T2.2** `internal/paths/paths.go`: `.tddmaster/state.json`, `{slug}/state.json`, `{slug}/progress/{phase}.json`.
- **T2.3** Mükerrer state'i ele: `completedTasks/decisions/transitionHistory` progress'ten türetilir, ayrı dosya yok.
- **T2.4** (Opsiyonel) eski `.tddmaster/.state/state.json` + `progress.json` → yeni şema migration aracı `tddmaster migrate`.

### Faz 3 — Manifest (JSON + düz)
- **T3.1** `internal/manifest/manifest.go`: yukarıdaki düz JSON şema struct'ları. `skipVerify/maxRefactorRounds/maxVerificationRetries/tddMode/importantTaskGate` **tepe seviye**.
- **T3.2** `internal/manifest/defaults.go`: `DefaultManifest()`.
- **T3.3** `phases.Enabled(manifest, specOverrides)`: global toggle + spec-bazlı `disabledSteps` birleştir/filtrele.
- **T3.4** Spec override mekanizması: `{slug}/state.json.disabledSteps[]` ile important-gate/refactor/verify step'lerini spec bazında kapat.

### Faz 4 — Phase katalogu
- **T4.1** `internal/phasecatalog/ids.go`: PhaseID/ModuleID/StepID sabitleri (yukarıdaki tablo).
- **T4.2** `internal/phasecatalog/catalog.go`: `Catalog[]` — 7 faz, modüller, stepler, `DefaultEnabled/ShowInUI` + her düğümde `Next` bağı (CoR zinciri) ve `Iterator` alanı. executing: `setup → cycle(Iterator="tasks") → finalize`.
- **T4.3** `internal/phasecatalog/config.go`: PhaseConfig/ModuleConfig/StepConfig.

### Faz 5 — Prompt Registry (birebir taşıma)
- **T5.1** `internal/promptregistry/keys.go`: InstructionKey + AgentRegistryKey sabitleri.
- **T5.2** `instructions_discovery.go` — discovery soruları + premise + mode promptları, tddmaster'dan **birebir**.
- **T5.3** `instructions_spec.go` — spec proposal/approve promptları.
- **T5.4** `instructions_execution.go` — executor/test-writer/verifier/planner body'leri + TDD phase instructions + behavioral kurallar (`strings.go`'dan birebir).
- **T5.5** `instructions_meta.go` — roadmap, protocol, gate, interactive.
- **T5.6** `instruction_registry.go`: `mergeInstructions([...])`, `Instructions` map.
- **T5.7** `completeness_test.go`: her StepDef'in referans verdiği InstructionKey'in registry'de var olduğunu doğrula (pentestmaster deseni).

### Faz 6 — Agent Registry + rename
- **T6.1** `internal/promptregistry/agent_registry.go`: `AgentSpec` + 4 agent (`tddmaster-test-writer/executor/verifier/planner`), body'ler birebir.
- **T6.2** Tüm bare `test-writer` referanslarını `tddmaster-test-writer` yap (registry + sync adapter + strings.go).
- **T6.3** Agent contract testi: her `DelegateAgent` key'inin registry'de tanımlı olduğunu doğrula.

### Faz 7 — Phase implementasyonları (step impls)
- **T7.1** `internal/engine/phases/phases.go`: `stepImpls` map + `Enabled()` filtre (pentestmaster deseni).
- **T7.2** `discovery.go`: listen + questions stepleri. Her soru → Prompt/Validate/Emit. Cevap anında `discovery.json`'a.
- **T7.3** `discovery_refinement.go`: premises + alternatives stepleri.
- **T7.4** `spec_proposal.go` + `spec_approved.go`: draft/approve + important bulk mark.
- **T7.5** **Generic iterator motorda** (TDD-agnostik): `engine.RegisterIterator("tasks", ...)`, `ModuleDef.Iterator`, cursor ilerletme + per-item reset. Motor `red/green/refactor` string'ini İÇERMEZ.
- **T7.5b** executing step impls = saf veri: `importantGate.plan`(→`tddmaster-planner`), `red.write-tests`(→`tddmaster-test-writer`), `green.implement`(→`tddmaster-executor`)+`verify`(→`tddmaster-verifier`), `refactor.apply`(→executor)+`recheck`(→verifier). Her biri sadece `DelegateAgent`+`InstructionKey`. skipVerify/gate disabled → manifest/override filtresiyle step skip (impl'de if-else yok).
- **T7.6** `blocked.go` + `completed.go`.

### Faz 8 — Spec & Context servisleri
- **T8.1** `internal/spec/`: spec parse + `spec.md` render (türev). tddmaster `internal/spec/` mantığı uyarlanır.
- **T8.2** `internal/context/`: cevapları prompta derleyen compile katmanı — tddmaster mantığı, ama state tek kaynaktan okunur.
- **T8.3** Discovery questions/concerns/extras tddmaster `internal/context/model/questions.go` + `concerns/` birebir.

### Faz 9 — CLI komutları
- **T9.1** `cmd/init.go`: scaffold `.tddmaster/` + `manifest.json` (interactive + `--non-interactive`).
- **T9.2** `cmd/spec.go`: `spec new/list/next [--answer]`, `task mark-important/undo`.
- **T9.3** `cmd/next.go`: **ince** — sadece `engine.Build → Next/Submit → JSON action yaz`. if-else state machine YOK.
- **T9.4** `cmd/config.go`: `config important-gate on|off|status`, `config skip-verify ...` → manifest/spec override yazar.
- **T9.5** `cmd/sync.go`: agent dosyalarını promptregistry'den üret (`.agents/`, tool adapter frontmatter).
- **T9.6** `cmd/undo.go`, `reopen` vb. recovery komutları.

### Faz 10 — Sync / agent dosya üretimi
- **T10.1** `internal/sync/`: promptregistry'deki AgentSpec'lerden tool-spesifik agent dosyaları üret (claude-code/antigravity/opencode/codex frontmatter). Body promptregistry'den, isim `tddmaster-*`.

### Faz 11 — Test & doğrulama
- **T11.1** Engine + state + manifest unit testleri yeşil.
- **T11.2** Uçtan uca akış testi: `init → spec new → next×N (discovery) → refinement → spec approve → executing(red/green/refactor + gate) → completed`. Her step'ten sonra ilgili `progress/*.json` diske yazıldığını assert et.
- **T11.3** Prompt completeness + agent contract testleri (T5.7, T6.3).
- **T11.4** Spec-bazlı toggle testi: bir spec'te `executing.refactor` ve `importantGate` disabled iken akışın o stepleri atladığını doğrula.
- **T11.5** Prompt birebir doğrulaması: refactor sonrası prompt metinleri tddmaster orijinalleriyle string-eşit (golden test).

---

## Kritik dosyalar

**Kaynak — pentestmaster (iskelet/mimari):**
- `internal/engine/{context,phase,module,step,action}.go` — motor (birebir kopya)
- `internal/phasecatalog/{ids,catalog,config}.go` — katalog deseni
- `internal/state/state.go` — tek-kaynak state deseni
- `internal/manifest/{manifest,defaults}.go` — JSON manifest + toggle
- `internal/promptregistry/{instruction_registry,agent_registry,keys}.go` — prompt merkezi
- `cmd/next.go` — ince CLI (state machine motor içinde)

**Kaynak — tddmaster (mantık + prompt, birebir alınacak):**
- `internal/sync/adapters/shared/{executor_prompt,test_writer,verifier_prompt,planner_prompt,agents_md}.go`
- `internal/context/service/tdd/instructions.go`, `internal/context/model/strings.go`
- `internal/context/service/meta/roadmap.go`, `internal/context/service/discovery/*.go`
- `internal/context/model/questions.go` — discovery soruları
- `internal/state/model/phase.go` — faz enum (referans)
- `.tddmaster/manifest.yml` — düzeltilecek hiyerarşinin kaynağı

**Hedef — kök dizin** (`/Users/pragmata/Projeler/tddmaster-refactor/`): yukarıdaki tüm `internal/`, `cmd/`, `main.go`, `go.mod`.

---

## Doğrulama (uçtan uca)

```bash
cd /Users/pragmata/Projeler/tddmaster-refactor
go build ./... && go test ./...

# akış dumanı
./tddmaster init --non-interactive
./tddmaster spec new "örnek feature"
./tddmaster spec <slug> next                       # discovery ilk soru döner
./tddmaster spec <slug> next --answer="..."        # her cevap sonrası:
cat .tddmaster/<slug>/progress/discovery.json      # → answered:true anında yazılmış

# spec-bazlı toggle
./tddmaster config important-gate off --spec <slug>
# executing'de importantGate.plan step'inin atlandığını gözle

# prompt birebir golden test
go test ./internal/promptregistry/...              # metinler tddmaster ile string-eşit
```

**Kabul kriterleri:**
1. Tek state kaynağı — `completedTasks/decisions/specDescription` hiçbir yerde mükerrer değil.
2. `cmd/next.go`'da phase'e dayalı if-else switch YOK; akış motor + katalog + manifest ile sürülüyor.
3. `manifest.json` (JSON), `skipVerify` tepe seviyede.
4. Important gate, TDD red/green/refactor, discovery soruları = step; spec bazında kapatılabilir.
5. Tüm promptlar `internal/promptregistry/` altında, metinler tddmaster orijinalleriyle birebir.
6. Üretilen agent isimleri `tddmaster-test-writer/executor/verifier/planner`.
7. Her step cevabı anında ilgili `progress/*.json`'a yazılıyor.
8. **Plug-and-play (motor/CLI'ye dokunmadan, sadece katalog+manifest):** (a) `green`↔`refactor` arası yeni `lint` step'i ekle → akış lint'i çalıştırsın; (b) `refactor` modülünü disable et → `green`'den sonra zincir `finalize`'a şeffaf aksın; (c) `importantGate` toggle çalışsın; (d) komple yeni iterating faz (setup→per-item body→finalize) eklenebilsin. Motor `red/green/refactor/gate` string'ini hiçbir yerde içermez.
9. **CoR/do-while:** executing fazı setup(1×) → cycle(her task) → finalize(1×) sırasıyla yürür; zincir düğümleri `Next` ile bağlı, disabled düğüm bypass.
