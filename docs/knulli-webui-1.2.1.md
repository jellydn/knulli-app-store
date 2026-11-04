# Knulli WebUI 1.2.1

## Status and scope

The owner authorized an **upstream-managed experimental** installation on 2026-11-03 after the security review in [the implementation thread](https://ampcode.com/threads/T-01a19f39-efaf-72c8-8855-79981979d3b9). It appears in the App category (`type: utility`). No device, controller, firmware-version or resolution test is claimed. Scope is AArch64 KNULLI with an existing Python 3 command; upstream source declares Python 3.7 or later. The service also needs Bash, `nice`, `nohup`, `stat`, and the standard KNULLI shell utilities. Optional screen capture and controller input depend on the device and its permissions.

The Store does not run `install.sh`, register a shared service, enable autostart, or start the server during installation. It installs a private service and a Tools launcher. Selecting **Knulli WebUI** in Tools starts the server; leaving Tools does not stop it.

## Configure before the first start

Edit `/userdata/system/knulli-webui/knulli_webui/config.json` before selecting the Tools launcher. Upstream defaults are **HTTP on every interface, port 8088, no authentication, uploads/deletion enabled, and remote input enabled**.

- Set `username` and `password`. Basic authentication over HTTP is not encrypted. For protected transport, bind `host` to `127.0.0.1` and use an authenticated encrypted tunnel.
- Set `allow_upload`, `allow_delete`, and `allow_remote_control` to `false` unless you need them. Set a finite `max_upload_mb` if you enable uploads; upstream uses zero for unlimited.
- Use only a trusted private network. Do not forward the port to the internet. A client on the network can supply the custom mutation header; that header is not authentication.
- Runtime writes are not sandboxed. Uploads accept executable content in Tools and can follow an existing system-directory symlink outside the ROM root. The Store does not fix these upstream behaviors.
- Use only ROMs, BIOS files and other content that you have the right to use and transfer. This app serves user-owned files, not a reviewed source of game downloads.

When listening on the LAN, open `http://<device-ip>:8088` in your browser. Credentials in `config.json` are plaintext; protect access to the device and its network shares.

## Stop, Repair and remove

Stop the private service before **Repair**, update, or removal:

```sh
/userdata/system/knulli-webui/services/knulli_webui stop
```

Use the same private command with `status`, `start` or `restart` as needed. The Store does not execute these commands for you. Repair restores pinned code, not runtime changes to other user files. Do not run Repair while the server is active.

Removal refuses an existing `/userdata/system/knulli-webui/knulli_webui/service.pid`, a legacy `/userdata/system/services/knulli_webui` file, or a matching manual boot reference in `custom.sh`. Stop and disable any independently installed legacy shared service before using this package; it uses a different code directory and `/var/run` PID, which the Store cannot inspect. Remove its service file only after it is stopped and disabled. Custom startup locations need manual review. Do not run upstream `install.sh --uninstall`: it recursively removes its app directory, including configuration.

The Store keeps `knulli_webui/config.json`, runtime-created files, and unowned logs at `/userdata/system/logs/knulli-webui.log` and `.log.1`. It does not remove ROMs, saves, BIOS, screenshots, gamelist backups, collections or frontend settings modified by the running app. Retained config includes credentials; removal is not credential erasure. A later installation requires **Manage existing** because those files remain; it keeps the retained config.

## Release, rights and Store writes

[Release 1.2.1](https://github.com/kubrick89/knulli_webui/releases/tag/1.2.1) resolves to commit `bc8f18d3fb1bb0d2a05f2dbdaf56ea247b4733fa`. Downloaded `knulli_webui_1.2.1.zip` is 1,133,508 bytes with SHA-256 `f39cf443f314ba8948c2ac9d4b069c81e33519aee491ba25b0fcc97c3cc9c555`, matching [GitHub asset 701485636](https://api.github.com/repos/kubrick89/knulli_webui/releases/assets/701485636). No separate publisher checksum was found. This trusts the publisher's GitHub account and GitHub, not independent publisher authentication. The release is not GitHub-immutable; the Store pins exact bytes and fails on replacement.

Archive review found 228 unique relative entries and 1,423,049 uncompressed bytes, without links, traversal, absolute paths or special files. Windows Zone.Identifier files and CPython 3.14 bytecode are present; the service invokes `webui.py`, not that bytecode. No native binary dependency is bundled.

No application-code license was found. The Store downloads directly from upstream under the authorized non-game experimental exception; it does not mirror or redistribute the asset. Do not invent an SPDX identifier. Bundled console icons retain their KyleBing CC BY 4.0 notice and modification credit. That notice does not license the application, and no dedicated Store artwork is copied.

Store writes are limited to:

- `/userdata/system/knulli-webui` for extracted files, with `config.example.json` mapped to the preserved `knulli_webui/config.json`;
- `/userdata/roms/tools/Knulli WebUI.sh` for the shell-quoted service-start wrapper;
- `/userdata/roms/tools/gamelist.xml` for the exact Tools entry;
- the Store's own journal and installed-state area.

One source-byte and final-hash-verified equal-length patch changes `APP_DIR` and `PIDFILE` in the private service at offset 225. Code is found relative to the service file, and the PID moves into the app's private userdata directory. The patched service SHA-256 is `bd922a7631d08a71c6b4d115c89c5b00710490ca407c170f9b37d648e96f66a9`. Everything else retains the publisher's bytes. Updates that change this script need manual review.

Runtime writes are broader: ROM/media upload and deletion, gamelists and backups, saves/state images, BIOS uploads, screenshots, collections/frontend settings and backups, service PID/logs, and controller event devices. Screen capture reads framebuffer/device data and may execute a locally configured command. Store integrity controls do not confine these actions.

## Verification on 2026-11-03

- A temporary integration test used the actual pinned publisher ZIP through `installer.Manager` in a fixture root: installation, patched-file health, Repair, preserved configuration, each declared removal guard, and removal passed. It never executed a launcher or installer. The temporary test was removed after validation; the retained lifecycle regression uses a small synthetic inventory.
- The retained lifecycle regression also damages managed Python source before Repair, verifies restored code and retained credentials, checks that removal leaves a sample save intact, and verifies that a later **Manage existing** keeps the retained configuration. Service-state and shared-startup guards are checked independently of the manifest's list.
- A separate Python 3.11.6 check loaded the actual packaged source and started its HTTP server on loopback with fixture paths and remote input disabled. `/api/config` returned 200 with version 1.2.1. After configuring fixture credentials, it returned 401 without credentials and 200 with them. The server was stopped and its socket closed. This is source-server startup evidence, not a KNULLI service or hardware test.
- Service-patch verification checked original bytes, the final SHA-256, Bash syntax, and package-relative app/PID paths, including a path with spaces. It did not run the full service or exercise its process shutdown on KNULLI.
- `make check`, `go test -race -tags sdl ./...`, manifest validation, and `make catalogue catalogue-check` passed. The generated index contains eleven packages; both catalogue builds matched byte for byte (113,988 bytes). `sh -n scripts/desktop-fixture.sh` and `git diff --check` also passed.

No exact-device result, real ROM/save mutation, framebuffer capture, controller input, or shared-service migration is claimed. The visible network and license warnings remain necessary.
