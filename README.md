# Photon


Unofficial Google Photos Desktop GUI Client

Photon is a fork of [gotohp](https://github.com/xob0t/gotohp) by xob0t, with many additions (browser library, in-app sign-in, watched folders, upload history and more)

- Unlimited uploads (can be disabled)
- File and folder selection interface
- Sign in with a Google account directly in the app (no phone needed), or with mobile app credentials
- Credential management for multiple accounts
- Album targeting: no album, a new album, or an existing album (listed with its item count)
- Activity tab with real-time upload progress, speed and ETA
- Configurable upload threads, lowered automatically when the connection is struggling
- Individual files or directories uploads, with optional recursive scanning
- Skips files already present in your account
- Local upload history: already-uploaded files are skipped offline and
  unchanged files are not re-hashed
- EXIF capture date support: photos are committed with the date they were
  taken, not the file date
- Failed files are retried once automatically 10 seconds after the batch ends (optional)
- Uploads started while another batch is running are queued and run one after another
- "Move originals to the Recycle Bin" (Delete from host) moves files to the Recycle Bin / Trash instead of deleting them. If the trash can't be used (e.g. no `gio` on Linux, or a file on another macOS volume) the file is kept and a warning is shown
- Pause, resume, and cancel uploads; an upload batch interrupted by closing the app can be resumed on the next launch
- Optional proxy, and "Keep running in the tray" so closing the window leaves uploads going
- Watched folders: files that appear in a watched directory upload automatically
- System notification when all uploads are done
- Browser library: browse and upload to your library from any browser on your network
- Configurable, persistent upload settings (stored in "%system config path%/Photon/Photon.config")  
   By default it lives in your user config folder. For a portable setup, create a `data` folder next to the executable containing an empty `Photon.config`; Photon then keeps its config, upload history and browser library data (`history.db`, `library-web.config`, `library-data/`) in that `data` folder. A `Photon.config` placed directly next to the executable (older versions) is moved into `data` automatically.  
   Upgrading from the old "gotohp" name: your existing config and upload history are moved to the new names automatically on first launch.

## Albums

On the Home tab, **Destination** decides where uploads go:

- **No album** - files go to your library only.
- **New album** - creates an album with the name you type.
- **Existing** - adds to an album you pick from the list. Albums are shown as
  `Name (item count)`, for example `Fav (700)`. The list is cached per account
  so it is there as soon as the app opens; use the refresh button to reload it.

Uploading to an album by name adds to an existing album with exactly that name
if one exists (and has room), instead of creating another album of the same
name. An album key (`AF1Qip...`) is also accepted. An album holds up to 20,000 items; larger batches
are split across several albums.

## Activity

Once an upload starts, the Activity tab shows every file with its status,
per-thread progress, speed and ETA. From there you can pause, resume or cancel
the batch, skip a file in progress, or remove a queued file. Starting another
upload while one is running adds it to the queue.

**Upload threads** is a ceiling. If a file needs retrying, Photon lowers the
number of files sent at once (down to 1) and raises it again, up to your
setting, after 5 uploads in a row go through cleanly. When a batch (or the
whole queue) finishes you get a system notification with the uploaded, failed
and skipped counts, if notifications are on. If the app is
closed in the middle of a batch, Home offers to resume the unfinished files the
next time it opens.

## Watched folders

Watched folders upload files on their own: pick a folder (and, optionally, the
album targeting selection from the Home tab), and any file that appears in it —
including subdirectories — is uploaded a few seconds after it stops changing.

- Files that already exist in a watched folder when the app starts are left
  alone; only newly appeared or changed files are picked up.
- Watched uploads always skip files already in your library (the hash check),
  even with "Upload duplicates too" enabled, so restarting the app never re-uploads a
  watched folder.
- Each watcher can be paused, retargeted to a different album, or removed
  from the Home tab.

## File types

Supported extensions: avif, bmp, gif, heic, heif, ico, jpg, jpeg, png, tif, tiff,
webp; RAW (cr2, cr3, nef, arw, orf, raf, rw2, pef, sr2, dng); and video (3gp, 3g2,
asf, avi, divx, m2t, m2ts, m4v, mkv, mmv, mod, mov, mp4, mpg, mpeg, mts, tod, wmv, ts).

Files are matched by extension (turn on "Try unsupported file types" in Settings to skip this filter). For the ambiguous ones (`.ts`, `.mod`, `.tod`) the file's first bytes must also look like video, so TypeScript sources and `go.mod` files in a scanned folder are not uploaded. Folders that cannot be read are reported as a warning instead of being skipped silently.

To skip a folder by name while scanning (and in watched folders), add `exclude_pattern: <folder name>` to `Photon.config`. It matches a single folder name exactly, and there is no setting for it in the app.

## Upload history

Every completed upload (or hash-check hit) is remembered in a local database
next to the config (`history.db`), keyed by file path, size and modification
time. This powers two things: files you have already uploaded are skipped
without any network request, and unchanged files are not re-hashed from disk.
It can be cleared from Settings → Data. With "Delete from host" on, history alone is
never trusted before a local file is moved to the trash: the library is
checked first, so an item you removed from Google Photos is re-uploaded
instead of your only copy being deleted.

## Upload dates

The timestamp a photo is committed with is resolved in this order: the EXIF
capture date (JPEG/TIFF, when present and "Use the date the photo was taken" is enabled; the camera's UTC offset is
used when the file records one, otherwise this computer's timezone), then a timestamp parsed
from the filename (`20240131_235959`, `2024-01-31 23:59:59`, `2024-01-31`, or a 13-digit Unix millisecond timestamp) (if "Read the date from the file name" is enabled), then the
file's modification time.


## Browser library

Photon can serve your Google Photos library and albums to a web browser, on this computer or on any other device on your local network (browse, watch videos, manage albums, upload). It is the Photon web interface, built into the app and sharing its accounts and upload settings.

Turn it on in **Settings → Browser library**:

- **Turn on the browser library** starts the server; turning it off stops it.
- **Allow other devices on my network** listens on all network interfaces so phones and other computers on the same Wi-Fi/LAN can open it. It is **off by default**, which means it only answers on this computer.
- **Port** is the port to listen on (default `8787`). Press **Apply** to restart on the new port. If the port is taken, the status shows the error.
- Below the settings, the **Running** box lists the addresses to open (for example `http://localhost:8787` and `http://192.168.1.23:8787`) with copy and open buttons.

What you can do in the browser:

- Browse **All**, **Photos**, **Videos**, **Favorites**, **Archive**, **Trash** and each album, with search and sorting.
- Play videos (streamed through Photon), download originals, and view thumbnails (cached on disk, 300 MB by default).
- Favorite, archive, trash, restore, permanently delete and edit captions.
- Create, rename and delete albums, and add or remove photos.
- Upload files, with pause, resume, cancel and skip controls.
- Keep track of what you have seen: the **Unwatched** and **Watched** menus in the sidebar sort your videos by whether you have played them (see below).
- Find likely duplicates: items whose file names share the same first word (for example `IMG_0001` and `IMG_0001 (1)`) are grouped together.
- Switch accounts and change the upload settings (some settings can only be changed on the computer running Photon, see below).

The library is synced in the background every 2 minutes (**Sync now** forces it).
You can change this in `library-web.config` with `librarySyncSec` (minimum 15,
a negative value turns auto-sync off) and the thumbnail cache size with `thumbCacheMB`.

### Watched and Unwatched videos

Every video starts in **Unwatched**. As soon as you have actually played at least **2%** of a video's duration (jumping ahead with the seek bar counts too), it moves to **Watched**. The count next to each menu updates right away, and the open video stays where it is until you close the viewer.

- You can also mark videos by hand: use the eye button in the viewer, or select videos and press **Mark watched** / **Mark unwatched**.
- The state is stored by Photon in `watched.json` next to the library cache (`library-data/`, or your `data` folder in a portable setup), separately for each Google account. It is the same for every browser and device that opens this library, and it survives re-syncs. If you run Photon on several computers, copy `watched.json` between them (or point them at the same data folder) to share it.
- Nothing is written to Google Photos: your files, albums, favorites and captions are never touched. A video you added to an album is still the same library video and is listed once. Videos in the Trash are not listed.

First time only: open the `localhost` address in a browser **on the computer running Photon** and create the web login. Creating that login is blocked from other devices, so nobody on your network can claim it first. After that, sign in from any device (passwords must be at least 8 characters, and a login stays valid for 30 days), then press **Sync now** to load your library.

Notes:

- Sessions from other devices can browse, upload (by sending files) and manage albums, but cannot upload by host path, and cannot change "Delete from host", the file-type filter or the proxy; those can only be changed from the computer running Photon. Failed logins are rate-limited per device.
- Uploading to an album by **name** adds to an existing album with exactly that name if one exists (and has room), instead of creating another album of the same name.
- The connection is plain HTTP on your network. Only enable network access on networks you trust, and allow the port in your firewall if other devices can't connect.
- The web login and library cache live next to the app config (`library-web.config` and `library-data/`). Accounts and upload settings are shared with the desktop app, so adding or removing an account in either place shows up in both.


## Adding a Google account

You only need to do this once per account. Photon needs a Google Photos mobile
credential; there are three ways to get one.

### Option 1 - Embedded setup. Easiest, no phone or root

1. On the **Home** tab, open the account sign-in (**Sign in to Google Photos**).
2. Click **Open Google sign-in**, sign in with the account you want to add and click **I agree**.
   The page may keep loading forever; that is expected.
3. Open your browser's developer tools and go to Application (or Storage) → Cookies → `accounts.google.com`.
4. Copy only the value of the `oauth_token` cookie.
5. Paste it into Photon and click **Connect account**.

Photon exchanges the token for a long-lived credential, checks that it works
and saves the account. The `oauth_token` is single use, so if it fails, sign in
again and copy a fresh one.

### Option 2 - ReVanced. No root required

1. Install Google Photos ReVanced on your android device/emulator.
   - Install GmsCore [https://github.com/ReVanced/GmsCore/releases](https://github.com/ReVanced/GmsCore/releases)
   - Install patched apk [https://github.com/RookieEnough/Morphe-AutoBuilds/releases](https://github.com/RookieEnough/Morphe-AutoBuilds/releases) or patch it yourself
2. Connect the device to your PC via ADB.
3. Open the terminal on your PC and execute

   Windows

   ```cmd
   adb logcat | FINDSTR "auth%2Fphotos.native"
   ```

   Linux

   ```shell
   adb logcat | grep "auth%2Fphotos.native"
   ```

4. If you are already using ReVanced - remove Google Account from GmsCore.
5. Open Google Photos ReVanced on your device and log into your account.
6. One or more identical GmsCore logs should appear in the terminal.
7. Copy text from `androidId=` to the end of the line from any log.
8. That's it! 🎉

### Option 3 - Official apk. Root required

<details>
  <summary><strong>Click to expand</strong></summary>

1. Get a rooted android device or an emulator.
2. Connect the device to your PC via ADB.
3. Install [HTTP Toolkit](https://httptoolkit.com)
4. In HTTP Toolkit, select Intercept - `Android Device via ADB`. Filter traffic with

   ```text
   contains(https://www.googleapis.com/auth/photos.native)
   ```

   Or if you have an older version of Google Photos, try

   ```text
   contains(www.googleapis.com%2Fauth%2Fplus.photos.readwrite)
   ```

5. Open Google Photos app and login with your account.
6. A single request should appear.  
   Copy request body as text.
7. Add that credential string in Photon.
8. If Photon asks for a token binding key, keep the rooted device connected and click `Read from ADB`. Photon will read the account's `lstBindingKeyAlias` from Android AccountManager and save it into the credential.

#### Troubleshooting

- **No Auth Request Intercepted**
  1. Log out of your Google account.
  2. Log in again.
  3. Try `Android App via Frida` interception method in HTTP Toolkit.
- **Token binding key not found**
  1. Make sure the same Google account is present on the connected device.
  2. Make sure root is available to ADB.

</details>

## Build

Photon is a [Wails v3](https://v3alpha.wails.io/getting-started/installation/)
app: a Go backend with a Vue 3 + Tailwind frontend.

Requirements:

- Go (see `go.mod` for the version)
- Node.js with npm (the build tasks use `npm`)
- The `wails3` CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- Linux only: gcc, pkg-config, GTK 3 and WebKitGTK 4.1 development packages
- Windows installer only: [NSIS](https://nsis.sourceforge.io/) (`makensis`)

From the project root:

| Task | Windows | Linux |
| --- | --- | --- |
| Production build | `build-windows.bat` | `./build-linux.sh` |
| Packages | `wails3 task windows:package` | `./build-linux.sh package` (AppImage, .deb, .rpm, Arch) |
| Development mode | `wails3 task dev` | `wails3 task dev` |

Output goes to `bin/`. Run `task lint` / `task fmt` for linting and formatting
(needs `gofumpt` and `golangci-lint`).
