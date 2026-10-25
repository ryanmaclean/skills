#!/usr/bin/env nu
# NetBSD aarch64 QEMU VM standup on Apple Silicon
# Usage: nu standup-vm.nu [--ram 8G] [--disk 20G] [--ssh-port 2222] [--name nb] [--dir /tmp/netbsd-vm]
#
# Automates: image download, QEMU launch in screen, wait for boot,
# SSH key injection, DNS config, PATH setup, package bootstrap.
#
# Requires: qemu-system-aarch64, screen (brew install qemu screen)

def main [
    --ram: string = "8G"         # VM RAM
    --disk: string = "20G"       # Disk image size
    --ssh-port: int = 2222       # Host port forwarded to VM SSH
    --name: string = "nb"        # screen session name
    --dir: string = "/tmp/netbsd-vm"  # Working directory for image
    --skip-download: bool = false # Skip image download if already present
    --pubkey: string = ""        # SSH pubkey to inject (default: ~/.ssh/id_ed25519.pub)
] {
    let img_url = "https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/evbarm-aarch64/binary/gzimg/arm64.img.gz"
    let bios = "/opt/homebrew/share/qemu/edk2-aarch64-code.fd"
    let img_path = $"($dir)/arm64.img"
    let img_gz = $"($dir)/arm64.img.gz"

    # Resolve SSH pubkey
    let key = if $pubkey != "" {
        $pubkey
    } else if ("~/.ssh/id_ed25519.pub" | path expand | path exists) {
        open ("~/.ssh/id_ed25519.pub" | path expand) | str trim
    } else if ("~/.ssh/id_rsa.pub" | path expand | path exists) {
        open ("~/.ssh/id_rsa.pub" | path expand) | str trim
    } else {
        print "ERROR: No SSH pubkey found. Pass --pubkey or create ~/.ssh/id_ed25519.pub"
        exit 1
    }

    # Preflight
    print "=== NetBSD VM Standup ==="
    for cmd in ["qemu-system-aarch64", "screen"] {
        if (which $cmd | is-empty) {
            print $"ERROR: ($cmd) not found. brew install qemu screen"
            exit 1
        }
    }
    if not ($bios | path exists) {
        print $"ERROR: UEFI firmware not found at ($bios)"
        exit 1
    }

    # Check for existing screen session
    let existing = (^screen -ls | lines | where {|l| $l =~ $name})
    if ($existing | length) > 0 {
        print $"WARNING: screen session '($name)' already exists:"
        print ($existing | str join "\n")
        print "Kill it first: screen -S nb -X quit"
        exit 1
    }

    # Working directory
    mkdir $dir
    print $"Working directory: ($dir)"

    # Download image
    if $skip_download and ($img_path | path exists) {
        print "Skipping download — image exists"
    } else {
        print "Downloading NetBSD 10.1 aarch64 image..."
        ^curl -L -o $img_gz $img_url
        if $env.LAST_EXIT_CODE != 0 { print "ERROR: download failed"; exit 1 }
        print "Decompressing..."
        ^gunzip -f $img_gz
        print $"Resizing disk to ($disk)..."
        ^qemu-img resize $img_path $disk
    }

    # Launch QEMU in screen
    print $"Launching QEMU in screen session '($name)'..."
    let qemu_cmd = $"qemu-system-aarch64 -M virt -accel hvf -cpu host -smp 4 -m ($ram) -bios ($bios) -drive if=virtio,file=($img_path) -device virtio-rng-pci -device virtio-net-pci,netdev=net0 -netdev user,id=net0,hostfwd=tcp::($ssh_port)-:22 -nographic"
    ^screen -dmS $name sh -c $qemu_cmd
    if $env.LAST_EXIT_CODE != 0 { print "ERROR: screen launch failed"; exit 1 }
    print "QEMU started in screen. Waiting for boot..."

    # Wait for boot (poll SSH)
    print $"Polling SSH on localhost:($ssh_port)..."
    mut booted = false
    for attempt in 1..60 {
        sleep 5sec
        let result = (do { ^ssh -o StrictHostKeyChecking=no -o ConnectTimeout=3 -o BatchMode=yes -p $ssh_port root@localhost "echo ALIVE" } | complete)
        if $result.stdout =~ "ALIVE" {
            $booted = true
            print $"VM booted after ($attempt * 5)s"
            break
        }
        if ($attempt mod 6) == 0 {
            print $"Still waiting... (($attempt * 5)s)"
        }
    }

    if not $booted {
        # SSH with key might not work yet — try injecting via screen console
        print "SSH not responding with key auth. Injecting setup via console..."
        let cmds = [
            "echo nameserver 8.8.8.8 > /etc/resolv.conf"
            "export PATH=/usr/sbin:/usr/bin:/bin:/usr/pkg/bin:/usr/pkg/sbin:$PATH"
            $"mkdir -p /root/.ssh && echo '($key)' > /root/.ssh/authorized_keys && chmod 700 /root/.ssh && chmod 600 /root/.ssh/authorized_keys"
            "echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config"
            "echo 'PubkeyAuthentication yes' >> /etc/ssh/sshd_config"
            "/etc/rc.d/sshd restart"
            "passwd -d root"
        ]
        for cmd in $cmds {
            # Send command via screen stuff
            ^screen -S $name -p 0 -X stuff $"($cmd)\r"
            sleep 2sec
        }
        print "Console injection done. Waiting 15s for sshd restart..."
        sleep 15sec

        # Retry SSH
        for attempt in 1..12 {
            let result = (do { ^ssh -o StrictHostKeyChecking=no -o ConnectTimeout=3 -p $ssh_port root@localhost "echo ALIVE" } | complete)
            if $result.stdout =~ "ALIVE" {
                $booted = true
                print "SSH now working after console injection"
                break
            }
            sleep 5sec
        }
    }

    if not $booted {
        print "WARNING: SSH still not responding. VM may need manual login."
        print $"Console: screen -r ($name)"
        print $"SSH: ssh -p ($ssh_port) root@localhost"
        exit 1
    }

    # Bootstrap the VM
    print "=== Bootstrapping VM ==="
    let ssh_cmd = $"ssh -o StrictHostKeyChecking=no -p ($ssh_port) root@localhost"

    # DNS
    print "Configuring DNS..."
    ^sh -c $"($ssh_cmd) 'echo nameserver 8.8.8.8 > /etc/resolv.conf'"

    # PATH in profile
    print "Setting PATH in /root/.profile..."
    ^sh -c $"($ssh_cmd) 'grep -q usr/pkg /root/.profile 2>/dev/null || echo \"export PATH=/usr/sbin:/usr/bin:/bin:/usr/pkg/bin:/usr/pkg/sbin:\\$PATH\" >> /root/.profile'"

    # PKG_PATH in profile
    ^sh -c $"($ssh_cmd) 'grep -q PKG_PATH /root/.profile 2>/dev/null || echo \"export PKG_PATH=https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.1/All\" >> /root/.profile'"

    # Install base packages
    print "Installing pkgin, git, curl..."
    ^sh -c $"($ssh_cmd) 'export PATH=/usr/sbin:/usr/bin:/bin:/usr/pkg/bin:/usr/pkg/sbin:$PATH && export PKG_PATH=https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.1/All && pkg_add pkgin git-base curl'"

    print ""
    print "=== NetBSD VM Ready ==="
    print $"  Screen session: ($name)"
    print $"  SSH: ssh -p ($ssh_port) root@localhost"
    print $"  Console: screen -r ($name)"
    print $"  Image: ($img_path)"
    print ""
    print "Next steps:"
    print "  - Install Rust: pkg_add rust  (or use prebuilt binary)"
    print "  - Clone irclaw: git clone http://10.0.2.230:3001/studio/irclaw.git"
    print "  - Build: cd irclaw && cargo build --release"
}
