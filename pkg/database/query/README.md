## User

- `CreateUser`: Menambahkan user baru ke tabel dengan data yang diberikan.
- `GetUserByID`: Mengambil user berdasarkan ID jika belum dihapus (soft-delete).
- `GetUserByEmail`: Mengambil user berdasarkan email jika belum dihapus.
- `GetActiveUsers`: Mengambil semua user aktif (tidak dihapus).
- `GetTrashedUsers`: Mengambil semua user yang dihapus (soft-delete).
- `SearchUsers`: Melakukan pencarian berdasarkan nama depan, nama belakang, atau email dengan dukungan paginasi.
- `CountActiveUsers`: Menghitung jumlah user aktif (tidak dihapus).
- `TrashUser`: Menandai user sebagai terhapus (soft-delete).
- `RestoreUser`: Mengembalikan user yang sudah dihapus (soft-delete).
- `UpdateUser`: Memperbarui data user berdasarkan ID.
- `DeleteUserPermanently`: Menghapus user secara permanen dari tabel.
- `SearchUsersByEmail`: Melakukan pencarian khusus berdasarkan email.

------------------------------------------------------------------------------

## Card

- `CreateCard`: Menyisipkan kartu baru dan mengembalikan rekaman yang dibuat.
- `GetCardByID`: Mengambil kartu berdasarkan ID-nya, tidak termasuk kartu yang dibuang.
- `GetActiveCards`: Mengambil semua kartu yang tidak dibuang.
- `GetTrashedCards`: Mengambil semua kartu yang dibuang.
- `GetCards`: Mencari kartu dengan filter (misalnya, card_number, card_type, card_provider) dan mendukung penomoran halaman.
- `TrashCard`: Menandai kartu sebagai dibuang dengan menyetel kolom deleted_at.
- `RestoreCard`: Mengembalikan kartu yang dibuang dengan menyetel kolom deleted_at ke NULL.
- `UpdateCard`: Memperbarui detail kartu dan menyegarkan kolom updated_at.
- `DeleteCardPermanently`: Menghapus kartu secara permanen dari basis data.
- `GetCardsByUserID`: Mengambil semua kartu yang terkait dengan pengguna tertentu.
- `GetCardByCardNumber`: Mengambil kartu menggunakan nomor kartunya.
----------------------

## Merchant

- `CreateMerchant`: Menyisipkan rekaman pedagang baru dan mengembalikan entri yang dibuat.
- `GetMerchantByID`: Mengambil pedagang berdasarkan ID jika tidak dibuang.
- `GetActiveMerchants`: Mencantumkan semua pedagang aktif (tidak dibuang).
- `GetTrashedMerchants`: Mencantumkan semua pedagang yang dibuang.
- `GetMerchants`: Mencari pedagang berdasarkan nama, api_key, atau status dengan penomoran halaman.
- `TrashMerchant`: Menghapus sementara pedagang dengan menyetel kolom deleted_at.
- `RestoreMerchant`: Mengembalikan pedagang yang dibuang dengan mengosongkan kolom deleted_at.
- `UpdateMerchant`: Memperbarui detail pedagang.
- `DeleteMerchantPermanently`: Menghapus rekaman pedagang secara permanen dari basis data.
- `GetMerchantByApiKey`: Mengambil pedagang berdasarkan kunci API uniknya.
- `GetMerchantByName`: Mengambil pedagang berdasarkan namanya.

## Saldo

- `CreateSaldo`: Memasukkan catatan saldo baru dan mengembalikan entri yang dibuat.
- `GetSaldoByID`: Mengambil saldo berdasarkan ID-nya jika tidak dibuang.
- `GetActiveSaldos`: Mencantumkan semua catatan saldo aktif (tidak dibuang).
- `GetTrashedSaldos`: Mencantumkan semua catatan saldo yang dibuang.
- `GetSaldos`: Mencari catatan saldo berdasarkan nomor kartu dengan penomoran halaman.
- `TrashSaldo`: Menghapus saldo sementara dengan menyetel kolom deleted_at.
- `RestoreSaldo`: Mengembalikan saldo yang dibuang dengan mengosongkan kolom deleted_at.
- `UpdateSaldo`: Memperbarui detail saldo, seperti card_number dan total_balance.
- `UpdateSaldoBalance`: Memperbarui saldo untuk saldo menggunakan nomor kartunya.
- `UpdateSaldoWithdraw`: Menangani penarikan saldo dengan mengurangi jumlah dan menyetel detail penarikan, memastikan saldo mencukupi.
- `DeleteSaldoPermanently`: Menghapus catatan saldo secara permanen dari database. 
- `GetSaldoByCardNumber`: Mengambil saldo berdasarkan nomor kartu yang terkait.
----------

## Transaction

- `CreateTransaction`: Menambahkan transaksi baru dan mengembalikan hasilnya.
- `GetTransactionByID`: Mendapatkan transaksi berdasarkan ID jika belum dihapus (soft-delete).
- `GetActiveTransactions`: Menampilkan semua transaksi yang aktif (tidak dihapus).
- `GetTrashedTransactions`: Menampilkan semua transaksi yang sudah dihapus (soft-delete).
- `GetTransactions`: Pencarian transaksi berdasarkan nomor kartu atau metode pembayaran dengan paginasi.
- `CountTransactionsByDate`: Menghitung jumlah transaksi berdasarkan tanggal tertentu.
- `CountAllTransactions`: Menghitung semua transaksi aktif.
TrashTransaction: Melakukan soft-delete pada transaksi.
- `RestoreTransaction`: Mengembalikan transaksi yang dihapus (soft-delete).
- `UpdateTransaction`: Memperbarui data transaksi.
- `DeleteTransactionPermanently`: Menghapus transaksi secara permanen.
- `GetTransactionsByCardNumber`: Mendapatkan transaksi berdasarkan nomor kartu.
- `GetTransactionsByMerchantID`: Mendapatkan transaksi berdasarkan ID merchant.

## Transfer

- `CreateTransfer`: Menambahkan transfer baru dan mengembalikan hasilnya.
- `GetTransferByID`: Mendapatkan transfer berdasarkan ID jika tidak dihapus (soft-delete).
- `GetActiveTransfers`: Menampilkan semua transfer yang aktif (tidak dihapus).
- `GetTrashedTransfers`: Menampilkan semua transfer yang dihapus (soft-delete).
- `GetTransfers`: Pencarian transfer berdasarkan pengirim/penerima dengan paginasi.
- `CountTransfersByDate`: Menghitung jumlah transfer berdasarkan tanggal tertentu.
- `CountAllTransfers`: Menghitung semua transfer aktif.
- `TrashTransfer`: Melakukan soft-delete pada transfer.
- `RestoreTransfer`: Mengembalikan transfer yang dihapus (soft-delete).
- `UpdateTransfer`: Memperbarui data transfer.
- `UpdateTransferAmount`: Memperbarui jumlah transfer tanpa mengubah data lainnya.
- `DeleteTransferPermanently`: Menghapus transfer secara permanen.
- `GetTransfersByCardNumber`: Mendapatkan transfer berdasarkan kartu sebagai pengirim atau penerima.
- `GetTransfersBySourceCard`: Mendapatkan transfer berdasarkan kartu pengirim.
- `GetTransfersByDestinationCard`: Mendapatkan transfer berdasarkan kartu penerima.

## Topup

- `CreateTopup`: Menambahkan data topup baru ke tabel dan mengembalikan data yang baru ditambahkan.
- `GetTopupByID`: Mengambil topup berdasarkan ID jika belum dihapus (soft-delete).
- `GetActiveTopups`: Mendapatkan semua topup yang aktif (tidak dihapus).
- `GetTrashedTopups`: Mendapatkan semua topup yang dihapus (soft-delete).
- `GetTopups`: Melakukan pencarian berdasarkan nomor kartu, nomor topup, atau metode topup dengan paginasi.
- `CountTopupsByDate`: Menghitung jumlah topup pada tanggal tertentu.
- `CountAllTopups`: Menghitung semua topup aktif (tidak dihapus).
- `TrashTopup`: Menandai data topup sebagai terhapus (soft-delete).
- `RestoreTopup`: Mengembalikan data topup yang telah dihapus (soft-delete).
- `UpdateTopup`: Memperbarui data topup berdasarkan ID.
- `UpdateTopupAmount`: Memperbarui jumlah topup tanpa mengubah data lainnya.
- `DeleteTopupPermanently`: Menghapus data topup secara permanen dari tabel.
- `GetTopupsByCardNumber`: Mendapatkan semua data topup berdasarkan nomor kartu.

## Withdraw

- `CreateWithdraw`: Menambahkan data withdraw baru ke tabel.
- `GetWithdrawByID`: Mengambil data withdraw berdasarkan ID jika data tidak dihapus (soft-delete).
- `GetActiveWithdraws`: Mengambil semua withdraw yang belum dihapus.
- `GetTrashedWithdraws`: Mengambil semua withdraw yang telah dihapus (soft-delete).
- `SearchWithdraws`: Mencari withdraw berdasarkan nomor kartu dengan pencarian berbasis LIKE dan mendukung paginasi.
- `CountActiveWithdrawsByDate`: Menghitung jumlah withdraw yang aktif berdasarkan tanggal.
- `TrashWithdraw`: Melakukan soft-delete pada withdraw dengan menandainya sebagai terhapus (menetapkan deleted_at).
- `RestoreWithdraw`: Mengembalikan withdraw yang telah dihapus.
- `UpdateWithdraw`: Memperbarui data withdraw berdasarkan ID.
- `DeleteWithdrawPermanently`: Menghapus withdraw secara permanen dari tabel.
- `SearchWithdrawByCardNumber`: Mencari withdraw berdasarkan nomor kartu.
