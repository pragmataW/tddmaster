# tddmaster — Yol Haritası

Bu dosya, planlanan beş iş kalemini mevcut kod tabanına dayanarak açıklar. Her madde; mevcut durum, hedef davranış, kapsam ve (varsa) karar verilmesi gereken açık soruları içerir.

İlgili faz akışı (`internal/phases/phases.go`):

```
settings → discovery → spec-proposal → refinement → analysis → execution → (rule-learning) → terminal
```

---

## 1. Spec yaşam döngüsü (geri dönüş / reopen / amend / archive)

### Mevcut durum

- Fazlar tek yönlüdür; motor bir kez ilerlediği faza geri dönemez.
- `refinement` fazı `refinement_approved` cevap anahtarıyla tek seferde kapanır (`internal/phases/refinement.go:43`).
- `analysis` fazındaki `return-to-refinement` seçeneği adının aksine faza geri dönmez; refine payload'unu **yerinde** uygulayıp audit'i yeniden koşturur (`internal/phases/analysis.go:29-31`, `applyEdit`).
- Spec'ler için hiçbir yaşam döngüsü komutu yoktur: iptal, arşiv, yeniden açma mümkün değil. `cmd/root.go` sadece `init/start/next/refine/visualize/rule` kayıtlı.
- `progress.json` tarafında `draft / executing / completed` durumları var (`internal/spec/model.go:37-40`) ama `cancelled`/`archived` gibi kullanıcı-taraflı durumlar yönetilmiyor.
- Tamamlanmış bir spec'e sonradan iş eklemenin (brownfield) yolu yok; OpenSpec'in "change proposal" ve spec-kit'in `/converge` komutu bu boşluğu dolduruyor.

### Hedef

Spec'lerin yaşam döngüsünü yöneten komutlar ve faz-geri-dönüş desteği:

- `tddmaster reopen <slug> --to=refinement|discovery` — spec'i önceki bir faza geri alır. Hedef fazdan sonraki fazların cevap anahtarları temizlenir (ör. refinement'a dönüşte `refinement_approved`, `analysis_*` anahtarları), faz işaretçisi geri sarılır, `spec.md` yeniden render edilir.
- `tddmaster amend <slug>` — tamamlanmış spec'i yeniden açar: refinement'a dönülür, yeni task'lar eklenir (mevcut `refine` payload'u ile), analysis yeniden koşar, execution'da **sadece yeni task'lar** çalışır (done olanlar korunur).
- `tddmaster cancel <slug> [--reason=...]` — spec'i terminal `cancelled` durumuna çeker; açık worktree varsa kullanıcıya raporlanır.
- `tddmaster archive <slug>` — tamamlanmış/iptal edilmiş spec'i arşivler (listeleme ve dashboard'dan düşürür, dosyaları korur).

### Kapsam / yapılacaklar

- Engine'e faz-geri-dönüş primitifi: faz bazlı "cevap anahtarı temizleme" haritası (her faz hangi anahtarları yazıyor bilgisi driver'larda zaten zımni; açık hale getirilecek).
- Execution sırasında reopen edge-case'leri: yarım kalmış `Exec` durumu olan task'lar, açık worktree'ler (`git worktree list` ile çakışma kontrolü), in-flight raporlar. Politika: reopen ancak execution'da aktif task yokken ya da `--force` ile.
- `amend` akışında task ID çakışmasını önlemek için `TaskSeq` üzerinden devam.
- `cancel`/`archive` durumlarının `status`/`list`/visualize tarafına yansıması.

### Açık sorular

- `reopen --to=discovery` discovery cevaplarını silsin mi, yoksa cevaplar korunup sadece yeniden onay mı istensin? (Öneri: koru + yeniden onay.)
- `archive` fiziksel taşıma mı (`.tddmaster/specs/_archive/`) yoksa sadece durum işareti mi? (Öneri: durum işareti, daha az yıkıcı.)

---

## 2. Token optimizasyonu — prompt düzeltmeleri

### Mevcut durum

Her `next` çağrısı, stage prompt'unu sıfırdan ve tam metin olarak üretir. Büyüklükler:

| Kaynak | Boyut | Not |
|---|---|---|
| `internal/prompts/templates/claude_md.tmpl` | ~17 KB | Her orkestratör oturumunda yüklenir |
| `internal/prompts/templates/verifier.tmpl` | ~13 KB | Her verifier spawn'ında |
| `internal/prompts/templates/test-writer.tmpl` | ~3.9 KB | |
| `internal/prompts/templates/auditor.tmpl` | ~3.6 KB | |
| `internal/prompts/templates/executor.tmpl` | ~2.9 KB | |

Ayrıca her stage prompt'una tekrar tekrar eklenen bloklar (`internal/engine/loop/stages.go`):

- `appendUserContext` — listen-first cevabının tamamı **her** stage prompt'una girer (verifier ve refactor dahil) (`loop_driver.go:70`).
- `appendACsAndECs` — tüm AC/EC listesi red, green, refactor, executor, verifier prompt'larının her birinde tekrarlanır.
- `appendApprovedPlan` + `appendRules` + `interactiveOptions`/`commandMap` metinleri her `ask`'ta yeniden gönderilir.
- `ExpectedInput.Example` JSON örnekleri her instruct aksiyonunda tam metin.

### Hedef

Aynı anlamı koruyarak orkestratör ve sub-agent token maliyetini ölçülebilir şekilde düşürmek.

### Kapsam / yapılacaklar

- **Şablon diyeti**: `claude_md.tmpl` ve `verifier.tmpl`'deki tekrar eden prose'un sıkıştırılması; AGENTS.md (~15 KB) ile çakışan bölümlerin tek kaynağa indirilmesi.
- **UserContext politikasının değişmesi**: listen-first bağlamı sadece anlam taşıyan stage'lere (gate, red) enjekte edilsin; verifier/refactor prompt'larından çıkarılsın (veya N karaktere kırpılsın).
- **Delta enjeksiyonu**: Aynı task'ın ardışık stage'lerinde değişmeyen bloklar (AC listesi, plan) tekrar gönderilmesin; bunun yerine `spec.md` yolu referans verilsin (sub-agent worktree'den ana repo yoluna okuyabilir) ya da "önceki stage ile aynı" işareti kullanılsın.
- **Ölçüm**: Her `next` çıktısındaki `instruction` alanının byte/char boyutunun loglanması (debug flag'i, örn. `TDDMASTER_DEBUG_PROMPT_SIZE=1`), böylece optimizasyon öncesi/sonrası karşılaştırılabilir olsun.
- **Kompakt JSON**: `interactiveOptions`/`commandMap`'te tekrarlanan `tddmaster next <slug> --answer=...` öneklerinin kısaltılması.

### Kabul kriteri

- Aynı fixture spec ile uçtan uca bir akışta toplam üretilen instruction byte'ı en az %30 küçülmeli; davranış/testler değişmemeli.

---

## 3. Paralel çakışma tahmini (file-overlap forecast)

### Mevcut durum

- DAG sadece **veri bağımlılığını** bilir (`internal/spec/dag.go`); iki paralel task aynı dosyaya dokunursa çakışma ancak merge anında ortaya çıkar ve orkestratör çözemezse task `merge-conflict` ile bloklanır.
- Analysis fazı linter'ı (`internal/spec/analysis_lint.go`) bugün sadece şunları denetler: AC'siz task, mükerrer kriter, dependency hataları. Dosya kesişimi kontrolü yok.
- Auditor prompt'una onaylı planların `touchedFiles` listesi zaten veriliyor (`internal/phases/analysis.go:92-97`) ama deterministik linter bunu kullanmıyor.
- Zamanlama problemi: planlar execution sırasında (gate stage'de) onaylanır — yani analysis fazında henüz `touchedFiles` çoğunlukla yoktur. Tasarım bunu hesaba katmalı.

### Hedef

Paralel çalışacak task'ların dosya kesişimlerini merge'den **önce** görünür kılmak; mümkünse otomatik serileştirmek.

### Kapsam / yapılacaklar

- **Refinement'a ipucu alanı**: Task'lara opsiyonel `touchedFilesHint []string` alanı (refine payload'u ile set edilebilir). Auditor/linter analysis fazında bu ipuçlarıyla kesişim kontrolü yapar.
- **Yeni lint kuralı** (`file-overlap`): Aralarında `dependsOn` sıralaması olmayan (yani paralel koşacak) task çiftlerinin dosya listeleri (hint veya onaylı plan) kesişiyorsa:
  - tam dosya kesişimi → `warn` (öneri: `dependsOn` ekle veya dosyaları böl),
  - aynı dosyada birden fazla paralel task + kesişen fonksiyon sinyali → `block` adayı.
- **Gate-time kontrolü**: Plan onaylandığı anda, `touchedFiles`'ı o anda bilinen diğer planlarla/hint'lerle karşılaştır; kesişim varsa gate cevabına `overlapWarning` alanı ekle (orkestratör kullanıcıya gösterir) veya task'ı dinamik olarak serileştir.
- **Merge sırası optimizasyonu**: Ready task setinden merge sırası belirlenirken kesişen task'ların önce/art arda merge edilmesi seçeneği.
- **Conflict hafızası**: `merge-conflict: <files>` ile bloklanan task'ların çakışan dosya kümeleri task metadata'sına yazılsın; yeniden planlamada (reopen/amend) bu bilgi kullanılsın.

### Açık sorular

- Kesişim durumunda otomatik `dependsOn` enjeksiyonu mu, yoksa sadece kullanıcıya karar bırakılması mı? (Öneri: kullanıcı kararı — motorun determinizm + "explicit > clever" ilkesiyle uyumlu.)

---

## 4. `tddmaster doctor`

### Mevcut durum

Proje ve spec sağlığını denetleyen hiçbir komut yok. Çökmüş oturumlardan kalan artıklar (orphan worktree'ler, yarım kalmış spec dizinleri, stale branch'ler) ancak elle fark ediliyor; paralel protokol "session başında `git worktree prune`" diyor ama bunu zorlayan bir şey yok.

### Hedef

Tek komutla proje + spec bütünlük denetimi: `tddmaster doctor [--spec <slug>] [--fix] [--json]`.

### Denetim listesi (koda dayalı)

- **Manifest**: `manifest.json` var mı, parse oluyor mu, `Normalize` sonrası geçerli mi (`internal/manifest/manifest.go`); seçili tool'ların adapter'ı registry'de var mı (`internal/adapter/registry.go`).
- **Tool iskeleti**: Seçili her tool için agent dosyaları yerinde mi (`.claude/agents/`, `.cursor/agents/`, `.codex/agents/`, `.opencode/agents/`), `CLAUDE.md`/`AGENTS.md` mevcut mu (`internal/paths/paths.go:24-46`).
- **Spec bütünlüğü** (her spec veya `--spec` ile tekil):
  - `state.json` / `progress.json` / `settings.json` parse ediliyor mu; faz değeri katalogda tanımlı mı.
  - Task ID'leri tek mi; DAG geçerli mi (`spec.ValidateDAG`); kriter ID'leri atanmış ve tek mi.
  - `minTestCoverage` sınırlar içinde mi (`Settings.ClampCoverage`).
  - `traceability.json` girdileri bilinmeyen task/kriter ID'sine referans veriyor mu.
  - `Exec.Worktree` referansları `git worktree list` ile örtüşüyor mu.
- **Artıklar**: `.tddmaster/worktrees/` altında sahipsiz dizinler; `tddmaster/<slug>/*` pattern'li stale branch'ler; `UpdatedAt`'i eski ve mid-stage kalmış `Exec` durumları (çökmüş oturum şüphesi).
- **Repo hijyeni**: `.tddmaster/worktrees/` `.gitignore`'da mı.

### Davranış

- Çıkış kodu: `0` temiz, `1` bulgu var (warning ayrımı `--strict` ile block yapılabilir).
- `--fix`: güvenli otomatik onarımlar — `git worktree prune`, coverage clamp, `spec.md` yeniden render, orphan worktree temizliği (onaylı).
- `--json`: CI/betik kullanımı için makine-okunur rapor.

---

## 5. Red → green → refactor döngüsünün skip-verify'lı / skip-verify'sız kesin davranışının kararlaştırılması

### Mevcut davranış (koddan doğrulanmış)

**TDD + verifier AÇIK (`skipVerifierEnabled=false`):**

```
red (test-writer: sadece fail eden testler, test KOŞTURMAZ)
  → green (executor: implementasyon, test koşturmaz)
    → verifier (suite'i koşturur, refactorNotes üretir, coverage ölçer)
      → notes varsa: refactor apply (executor, notları verbatim uygular)
        → verifier regression re-check → notes bitene/cap'e kadar döner
      → notes yoksa: task done
```

- Verifier fail → green'e geri (`failedACs` ile). Coverage yetersiz → red'e geri (`internal/engine/loop/stages.go:406-417`).
- Refactor bypass koruması var: notlar uygulanmadan refactor kapanamaz (`transition.go:62-68`).
- `maxRefactorRounds = 3` sabit (`loop_driver.go:22`).

**TDD + verifier KAPALI (`skipVerifierEnabled=true`):**

- Verifier **yine de green'de bir kez çağrılır** — refactor notes üretmek için (README davranış matrisi `on/true` satırı; `verifierStageImpl.Applies` TDD dalında skip kontrolü yapmaz, `stages.go:369-377`).
- Refactor apply tek submit'te kapanır: `refactorApplied: true` + `completed` aynı raporda (`execRefactorSkipVerifyText`); refactor sonrası **regression re-check yapılmaz**.

**Non-TDD:**

- `skipVerifier=false`: executor → verifier → fail ise executor'a geri.
- `skipVerifier=true`: executor tek başına, self-report ile kapanır.

### Tespit edilen sorunlar / belirsizlikler

1. **İsim-davranış çelişkisi**: `skipVerifierEnabled=true` TDD modunda verifier'ı gerçekten atlamıyor; green'de bir kez çalıştırıyor. Ayarın adı ile davranışı çelişik.
2. **Red'in "fail" kanıtı yok**: Test-writer testleri koşturamaz; green'e geçmeden önce testlerin gerçekten fail ettiğini kimse doğrulamaz. "Fail etmesi gereken test hiç yazılmamış/pass geçiyor" durumu sessizce green'e taşar.
3. **Skip-verify'da coverage drift**: Coverage tek ölçüm noktası green-verify; skip-verify modunda refactor sonrası re-check olmadığından refactor coverage'ı düşürebilir ve fark edilmez.
4. **Refactor notes kaynağı belirsiz**: Skip-verify modunda "verifier yok ama notes var" — notes üretimi ile verifier'ın atlanması kavramsal olarak çakışıyor; executor self-review'a mı geçmeli, yoksa notes tamamen kapanmalı mı?
5. **Fail döngü kapakları yok**: Verifier-fail → green döngüsü ve coverage-unmet → red döngüsü sınırsız (tek fren global iteration cap).
6. **Ayar granularitesi**: `skipVerifierEnabled` spec-geneli; "refactor notes istiyorum ama regression re-check istemiyorum" gibi ara kombinasyonlar ifade edilemiyor.

### Karar verilecek noktalar

| # | Soru | Seçenekler |
|---|---|---|
| 5a | Skip-verify TDD'de verifier gerçekten atlanmalı mı? | A) Tam atlama (green→done, notes yok) · B) Mevcut (green'de 1 kez) · C) `refactorNotesEnabled` diye ayrı ayara böl |
| 5b | Red→green geçişinde "testler fail" kanıtı kim versin? | A) Engine kendisi koştursun (deterministik, `go test` benzeri komut manifest'ten) · B) Verifier red-check stage'i · C) Test-writer'a "fail logu" raporlama zorunluluğu |
| 5c | Skip-verify refactor sonrası regression kontrolü? | A) Hiç · B) Executor'un kendisi test koşturup kanıt raporlasın · C) Engine deterministik koştursun |
| 5d | Döngü kapakları | `maxVerifyAttempts` (öneri 3) sonrası task `blocked` + kullanıcıya eskalasyon; coverage-red döngüsüne aynı kapak |
| 5e | `maxRefactorRounds` | Sabit 3 kalsın mı, settings'e mi taşınsın? |
| 5f | Non-TDD skip-verify | Mevcut self-report yeterli mi, yoksa en azından "testler geçiyor" kanıtı mı istensin? |

### Çıktı

Kararlar alındıktan sonra: davranış matrisi (README'deki tablo) güncellenecek, `stages.go`/`transition.go` kuralları kararlara göre değiştirilecek, yeni ayarlar settings fazına ve `spec.md`'ye yansıtılacak, mevcut davranış testleri (`loop_driver*_test.go`, `stage*_test.go`) yeni matrise göre revize edilecek.

---

## Önerilen uygulama sırası

1. **Madde 5** (davranış kararları) — kod değişikliklerinden önce karar gerekli; diğer işlerin test tabanını etkiler.
2. **Madde 2** (token optimizasyonu) — prompt'lar madde 5 ile zaten değişecek; tek seferde diyet.
3. **Madde 4** (`doctor`) — bağımsız, düşük riskli; sonraki maddelerin (özellikle 1) güvenlik ağı olur.
4. **Madde 1** (spec yaşam döngüsü) — en kapsamlı değişiklik; doctor'un `progress.json` denetimleri üzerine oturur.
5. **Madde 3** (paralel çakışma tahmini) — `touchedFilesHint` alanı ve gate-time kontrolleri, yaşam döngüsü (amend/reopen) ile etkileştiği için sonda.
