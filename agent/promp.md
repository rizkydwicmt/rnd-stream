Tujuan:
Buat API untuk tabel `tickets` yang menerima payload JSON (lihat contoh) dan mengembalikan rows dengan hasil transformasi `formulas`. Proses harus berjalan melalui middleware streaming (chunked HTTP JSON stream). Implementasi memakai Go (Golang) dan mengikuti best practice performa dan kebersihan kode.

Payload contoh (valid JSON):
{
"tableName": "tickets",
"orderBy": ["ticket_id","asc"],
"limit": 100,
"offset": 0,
"where": [
{"field":"status","op":"=","value":"open"},
{"field":"date_create","op":">=","value":"2025-01-01"}
],
"formulas": [
{
"params": ["ticket_id"],
"field": "id",
"operator": "",
"position": 2
},
{
"params": ["ticket_id","date_create"],
"field": "ticket_id_masked",
"operator": "ticketIdMasking",
"position": 1
}
]
}

Output contoh (valid JSON array streamed):
[
{ "ticket_id_masked": "TCK-*****345", "id": 12345 },
{ "ticket_id_masked": "TCK-*****346", "id": 12346 },
...
]

Requirement teknis (step-by-step):
1. Pelajari struktur codebase saat ini (domain, db layer, middleware). Cari contoh middleware stream yang sudah ada. Jika tidak ada, buat middleware streaming berbasis HTTP chunked JSON (flush per-record / per-batch).
2. Buat **domain/service baru** `tickets` (atau nama domain lain sesuai konvensi) untuk menangani logika ini.
3. **Binding & validasi payload**:
    - Gunakan struct typed untuk payload.
    - Validasi `tableName` (allowed whitelist), `orderBy` format, `limit` range, `offset >= 0`, `where` shape.
    - Validasi `formulas`: setiap item harus memiliki `params []string`, `field string`, dan `position int`. `operator` boleh kosong atau nama fungsi valid.
4. **Sorting formulas** berdasarkan `position` (ascending).
5. **Generate unique select list** (uniq params) dari semua `formulas.params` setelah disort. Urutkan deterministic (mis. berdasarkan posisi formula + param order).
6. **Query builder**:
    - Bangun SELECT clause secara dinamis dari uniq select list.
    - Support WHERE (dari payload) dengan parameter binding (no string interpolation).
    - Support ORDER BY, LIMIT, OFFSET dari payload.
    - Support raw SQL fragments jika benar-benar perlu, tapi tetap gunakan bound parameters.
7. **Select LIMIT 1 sampling**:
    - Lakukan `SELECT <cols> FROM table WHERE ... LIMIT 1` untuk membaca metadata/tipe.
    - *Rekomendasi tambahan:* jika DB driver memungkinkan, baca column types dari DB metadata (lebih aman daripada mengandalkan nilai baris pertama).
8. **Select COUNT**: jalankan query count (same where). Catat bahwa pada tabel sangat besar count bisa lambat; dokumentasikan trade-off.
9. **Select rows**: jalankan query utama dengan `LIMIT` dan `OFFSET`.
    - Gunakan batch processing (fetch N rows per iteration) jika `limit` besar, dan stream-kan tiap batch.
10. **Eksekusi & mapping**:
    - Eksekusi raw rows scan menggunakan generic mapper (lihat point generics).
    - Mapping hasil ke response JSON sesuai `formulas`: untuk setiap row, hitung field `field` dari `formulas` memanggil operator yang bersesuaian (mis. `ticketIdMasking(ticket_id, date_create)`).
    - Pastikan handling NULL menggunakan `github.com/guregu/null/v5`.
11. **Streaming & Memory**:
    - Gunakan middleware stream untuk menulis output bertahap ke client (chunked JSON).
    - Gunakan `jsonBufferPool` (sync.Pool) untuk menghindari GC churn pada encoding JSON.
    - Hindari alokasi per-row berlebih; reuse buffers/slices. Prioritaskan stack allocations (local arrays/slices with capacity) bila memungkinkan, dan reuse via pools.
12. **Raw query execution**:
    - Gunakan raw SQL builder (convenience) dan `Next()`/`Scan()` pattern untuk baca rows streaming dari DB driver.
13. **Generics**:
    - Buat generic helper untuk scan & mapping agar tidak duplikasi kode (mis. `ScanInto[T any](rows *sql.Rows) (T, error)`).
14. **Batch processing**:
    - Jika `limit` > threshold, fetch per-batch (mis. 1000) dan proses mapping tiap batch, stream hasil per-batch.
15. **Null handling**:
    - Gunakan `github.com/guregu/null/v5` untuk tipe nullable.
16. **Error handling & observability**:
    - Kembalikan error JSON pada client jika terjadi kesalahan fatal.
    - Logging terstruktur (request id, duration, memory usage per request).
17. **Tests**:
    - Unit test untuk: payload validation, formulas sorting, uniq select generation, query builder SQL string & bindings, mapping operator functions.
    - Integration test (with test DB) untuk full flow.
18. **Dokumentasi**:
    - Sertakan README kecil di domain baru yang menjelaskan payload, contoh response, trade-off, dan limitasi.

Non-functional constraints:
- Gunakan best practice Go (idiomatic, small functions, single responsibility).
- Prioritaskan readability dan maintainability.
- Hindari memory spike (gunakan pools, batch, streaming).
- Input values **must** be parameter-bound — jangan concatenate user values langsung ke SQL.

Tambahan teknis implementasi (snippets & rekomendasi):
- `jsonBufferPool := sync.Pool{ New: func() interface{} { b := make([]byte, 0, 4096); return &b } }`
- Gunakan DB driver yang support streaming rows (`database/sql` + driver).
- Untuk masking operator: implementasikan di Go sebagai function yang menerima typed params dan mengembalikan `null.String` atau `string`.
- Gunakan `context` pada semua DB call untuk cancelation/timeouts.

Hasil akhir yang diharapkan:
- Endpoint HTTP (mis. `POST /v1/tickets/stream`) yang menerima payload di atas dan mengembalikan stream JSON array sesuai contoh. Response harus valid JSON dan dikirim bertahap per-record atau per-batch tanpa memuat seluruh dataset ke memori.
