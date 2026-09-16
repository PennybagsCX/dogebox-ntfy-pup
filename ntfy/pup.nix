{ pkgs ? import <nixpkgs> {} }:

# ntfy notification hub pup: ntfy server v2.28.0 + frontdoor (onboarding page
# + reverse proxy) + health bridge. One container, three services.
# See README.md for the user-facing story.

let
  storageDirectory = "/storage";
  ntfyVersion = "2.28.0";

  # Official static release tarball (CGO-free Go binary, runs on any NixOS
  # container). Pinned per-arch: Dogebox images are amd64 or arm64.
  ntfyTarball =
    if pkgs.stdenv.hostPlatform.isAarch64
    then pkgs.fetchurl {
      url = "https://github.com/binwiederhier/ntfy/releases/download/v${ntfyVersion}/ntfy_${ntfyVersion}_linux_arm64.tar.gz";
      sha256 = "18a13411e315ba44781df222c432d27527fc089c2229a994c593beb9c1e247a0";
    }
    else pkgs.fetchurl {
      url = "https://github.com/binwiederhier/ntfy/releases/download/v${ntfyVersion}/ntfy_${ntfyVersion}_linux_amd64.tar.gz";
      sha256 = "881a1530e30e01f1dec202c7f41e1664e57edfb7844e73e21e345159ac3ea9b7";
    };

  ntfyBin = pkgs.stdenv.mkDerivation {
    name = "ntfy-${ntfyVersion}";
    src = ntfyTarball;
    # The tarball's root dir name carries the arch suffix
    # (ntfy_<ver>_linux_amd64 / ..._arm64); stay at the top and grab the
    # binary by name so one derivation covers both arches.
    sourceRoot = ".";
    dontBuild = true;
    installPhase = ''
      mkdir -p $out/bin
      install -m0755 ntfy_*/ntfy $out/bin/ntfy
    '';
  };

  # Main service: first-boot bootstrap (auth db, admin user, device token)
  # then exec the ntfy server. ntfy listens on 127.0.0.1:8098 inside the
  # container; frontdoor owns the exposed 8099.
  ntfy = pkgs.writeScriptBin "run.sh" ''
    #!${pkgs.stdenv.shell}
    # shellcheck shell=bash
    # (pkgs.stdenv.shell is bash; pipefail is load-bearing: a failed token
    # grep must abort the script rather than mint an empty device-token)
    set -euo pipefail

    # Private-by-default for everything this script and the ntfy server it
    # spawns write (auth db, cache, device token). frontdoor and bridge run
    # as separate service processes with their own umask.
    umask 077

    # Container HOME (/var/empty) is not writable; some ntfy operations
    # touch it, so point it at persistent storage.
    export HOME=${storageDirectory}
    STORAGE=${storageDirectory}
    mkdir -p $STORAGE/cache $STORAGE/data $STORAGE/etc $STORAGE/state $STORAGE/config

    PUP_IP="''${DBX_PUP_IP:-127.0.0.1}"
    BASE_URL="''${BASE_URL:-http://''${PUP_IP}:8099}"

    NTFY="${ntfyBin}/bin/ntfy"
    CFG="$STORAGE/etc/server.yml"

    # Declarative server config, regenerated every start. Auth on, deny-all,
    # attachments off, 12h retention, iOS instant delivery via the ntfy.sh
    # relay (ntfy.sh only ever sees hashed topic names).
    cat > "$CFG" <<EOF
    base-url: "$BASE_URL"
    listen-http: "127.0.0.1:8098"
    cache-file: "$STORAGE/cache/cache.db"
    cache-duration: "12h"
    auth-file: "$STORAGE/data/auth.db"
    auth-default-access: "deny-all"
    enable-login: true
    behind-proxy: true
    web-root: "disable"
    upstream-base-url: "https://ntfy.sh"
    EOF

    # ntfy's user/token/access commands take the config via NTFY_CONFIG_FILE
    # (they have no --config flag in 2.x) and need the auth database, which
    # the SERVER creates at startup. So: start the server first, wait for
    # the db, run first-boot bootstrap, then stay in the foreground.
    export NTFY_CONFIG_FILE="$CFG"
    "$NTFY" serve --config="$CFG" &
    NTFY_PID=$!

    tries=0
    until "$NTFY" access >/dev/null 2>&1; do
        tries=$((tries + 1))
        [ "$tries" -ge 60 ] && {
            echo "[run.sh] auth database did not become ready; exiting to retry"
            exit 1
        }
        ${pkgs.coreutils}/bin/sleep 1
    done

    # First boot only: generate admin + device users and a never-expiring
    # device token. All secrets stay in /storage; nothing is baked into the
    # image. Passwords are passed via NTFY_PASSWORD (2.x has no --password
    # flag); --ignore-exists makes re-runs after a failed attempt safe.
    if [ ! -s "$STORAGE/state/device-token" ]; then
        echo "[run.sh] first boot: generating users and device token"
        ADMIN_PW=$(${pkgs.coreutils}/bin/head -c 24 /dev/urandom | ${pkgs.coreutils}/bin/base64 | ${pkgs.coreutils}/bin/tr -d '/+=')
        DEV_PW=$(${pkgs.coreutils}/bin/head -c 24 /dev/urandom | ${pkgs.coreutils}/bin/base64 | ${pkgs.coreutils}/bin/tr -d '/+=')

        NTFY_PASSWORD="$ADMIN_PW" "$NTFY" user add --role=admin --ignore-exists dogebox-admin

        NTFY_PASSWORD="$DEV_PW" "$NTFY" user add --ignore-exists dogebox
        "$NTFY" access dogebox '*' rw

        # No --expires flag = the token never expires ("--expires 0" would
        # mint a token that dies within minutes). Piped straight to the file
        # — no intermediate token file on disk, not even briefly — and grep
        # finding no token aborts the script under pipefail, so an empty
        # device-token can't happen silently. Kept on one line: backslash
        # continuations inside nix indented strings are error-prone.
        "$NTFY" token add --label device dogebox | ${pkgs.gnugrep}/bin/grep -o 'tk_[A-Za-z0-9_-]*' | ${pkgs.coreutils}/bin/head -n1 > "$STORAGE/state/device-token"
        chmod 600 "$STORAGE/state/device-token"
        echo "[run.sh] bootstrap complete"
    fi

    wait $NTFY_PID
  '';

  frontdoor = pkgs.buildGoModule {
    pname = "frontdoor";
    version = "0.0.1";
    src = ./frontdoor;
    vendorHash = null;

    buildPhase = ''
      export GO111MODULE=off
      export GOCACHE=$(pwd)/.gocache
      go build -o frontdoor frontdoor.go qrcode_js.go page.go
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp frontdoor $out/bin/
    '';
  };

  bridge = pkgs.buildGoModule {
    pname = "bridge";
    version = "0.0.1";
    src = ./bridge;
    vendorHash = null;

    buildPhase = ''
      export GO111MODULE=off
      export GOCACHE=$(pwd)/.gocache
      go build -o bridge bridge.go
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp bridge $out/bin/
    '';
  };
in
{
  inherit ntfy frontdoor bridge;
}
