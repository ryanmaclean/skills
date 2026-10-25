#!/usr/bin/env nu

const devices = [
  {
    name: "qnas",
    ip: "10.0.2.35",
    ssh: "studio@10.0.2.35",
    datadog_filters: ["snmp_device:qnas", "device_ip:10.0.2.35", "device_ip:10.0.3.22", "snmp_device:10.0.2.35", "snmp_device:10.0.3.22"],
    aliases: ["qnas.local", "10.0.3.22", "10.0.12.2"]
  },
  {
    name: "qnas2",
    ip: "10.0.3.123",
    ssh: "studio@qnas2.local",
    datadog_filters: ["snmp_device:qnas2", "device_ip:10.0.3.123", "snmp_device:10.0.3.123"],
    aliases: ["qnas2.local", "qnas2"]
  }
]

def run_text [cmd: list<string>] {
  let out = do { ^($cmd.0) ...($cmd | skip 1) } | complete
  {
    command: ($cmd | str join " "),
    exit_code: $out.exit_code,
    stdout: ($out.stdout | str trim),
    stderr: ($out.stderr | str trim)
  }
}

def run_json [cmd: list<string>] {
  let result = run_text $cmd
  if $result.exit_code != 0 {
    return ($result | merge {json: null})
  }

  try {
    $result | merge {json: ($result.stdout | from json)}
  } catch {
    $result | merge {json: null}
  }
}

def print_step [label: string, result: record] {
  let status = if $result.exit_code == 0 { "OK" } else { "FAIL" }
  print $"[($status)] ($label)"
  if ($result.stdout | is-not-empty) { print $result.stdout }
  if ($result.stderr | is-not-empty) { print $result.stderr }
}

def print_auth_status [] {
  let result = run_json ["pup", "auth", "status"]
  let status = if $result.exit_code == 0 { "OK" } else { "FAIL" }
  print $"[($status)] pup auth status"

  if $result.json == null {
    if ($result.stdout | is-not-empty) { print $result.stdout }
    if ($result.stderr | is-not-empty) { print $result.stderr }
    return
  }

  print ({
    authenticated: $result.json.authenticated,
    site: $result.json.site,
    status: $result.json.status,
    expires_at: $result.json.expires_at
  } | to json --raw)
}

def print_metric_discovery [] {
  let result = run_json ["pup", "metrics", "list", "--filter=qnap", "--output=json"]
  let status = if $result.exit_code == 0 { "OK" } else { "FAIL" }
  print $"[($status)] QNAP metric discovery"

  if $result.json == null {
    if ($result.stdout | is-not-empty) { print $result.stdout }
    if ($result.stderr | is-not-empty) { print $result.stderr }
    return
  }

  print ({
    metric_count: ($result.json.metrics | length),
    metrics: $result.json.metrics
  } | to json --raw)
}

def print_metric_tags [] {
  let result = run_json ["pup", "metrics", "tags", "list", "snmp.qnap.disk.temperature", "--from=2h", "--output=json"]
  let status = if $result.exit_code == 0 { "OK" } else { "FAIL" }
  print $"[($status)] QNAP disk temperature tags"

  if $result.json == null {
    if ($result.stdout | is-not-empty) { print $result.stdout }
    if ($result.stderr | is-not-empty) { print $result.stderr }
    return
  }

  let tags = $result.json.data.attributes.tags
  print ({
    device_tags: ($tags | where {|tag| $tag =~ "^(device_|snmp_|host:|agent_host:|env:|role:|team:)" }),
    qnap_disk_ids: ($tags | where {|tag| $tag =~ "^qnap_disk_id:" }),
    qnap_disk_statuses: ($tags | where {|tag| $tag =~ "^qnap_disk_status:" })
  } | to json --raw)
}

def print_reachable_metric [filter: string] {
  let query = $"avg:snmp.device.reachable{($filter)} by {device_ip,device_hostname,host,snmp_device}"
  let result = run_json ["pup", "metrics", "query", "--query", $query, "--from", "2h", "--to", "now", "--output", "json"]

  if $result.json == null {
    return {
      filter: $filter,
      query: $query,
      status: "error",
      series_count: 0,
      error: ([$result.stdout, $result.stderr] | where {|text| $text | is-not-empty } | str join "\n"),
      series: []
    }
  }

  let summaries = $result.json.series | default [] | each {|series|
    let points = $series.pointlist | where {|point| ($point.1 | describe) != "nothing" }
    let last = if ($points | is-empty) { null } else { $points | last }
    {
      scope: $series.scope,
      points: ($points | length),
      latest_ts_ms: (if $last == null { null } else { $last.0 }),
      latest_value: (if $last == null { null } else { $last.1 })
    }
  }

  {
    filter: $filter,
    query: $query,
    status: $result.json.status,
    series_count: ($summaries | length),
    series: $summaries
  }
}

def print_reachable_metrics [device: record] {
  let summaries = $device.datadog_filters | each {|filter| print_reachable_metric $filter }
  let matched = $summaries | where series_count > 0 | get filter
  let errors = $summaries | where status == "error"
  let status = if ($errors | is-not-empty) {
    "FAIL"
  } else if ($matched | is-empty) {
    "WARN"
  } else {
    "OK"
  }
  print $"[($status)] Datadog reachable metric filters"
  print ({
    device: $device.name,
    matched_filters: $matched,
    filters: $summaries
  } | to json --raw)
}

def snmpwalk_cmd [ip: string] {
  let version = if "QNAP_SNMP_VERSION" in $env { $env.QNAP_SNMP_VERSION } else { "2c" }

  if $version == "3" {
    [
      "snmpwalk",
      "-v3",
      "-l", "authPriv",
      "-u", $env.QNAP_SNMP_USER,
      "-a", ($env.QNAP_SNMP_AUTH_PROTO? | default "SHA"),
      "-A", $env.QNAP_SNMP_AUTH_PASS,
      "-x", ($env.QNAP_SNMP_PRIV_PROTO? | default "AES"),
      "-X", $env.QNAP_SNMP_PRIV_PASS,
      $ip,
      "1.3.6.1.2.1.1"
    ]
  } else {
    [
      "snmpwalk",
      "-v2c",
      "-c", $env.QNAP_SNMP_COMMUNITY,
      $ip,
      "1.3.6.1.2.1.1"
    ]
  }
}

def can_snmpwalk [] {
  if (which snmpwalk | is-empty) {
    print "[SKIP] snmpwalk is not installed"
    return false
  }

  let version = if "QNAP_SNMP_VERSION" in $env { $env.QNAP_SNMP_VERSION } else { "2c" }
  if $version == "3" {
    let required = ["QNAP_SNMP_USER", "QNAP_SNMP_AUTH_PASS", "QNAP_SNMP_PRIV_PASS"]
    let missing = $required | where {|name| $name not-in $env }
    if ($missing | is-not-empty) {
      print $"[SKIP] missing SNMPv3 env vars: (($missing | str join ', '))"
      return false
    }
  } else if "QNAP_SNMP_COMMUNITY" not-in $env {
    print "[SKIP] QNAP_SNMP_COMMUNITY is not set"
    return false
  }

  true
}

def check_device [device: record, do_snmpwalk: bool] {
  print $""
  print $"## ($device.name) (($device.ip))"

  print_step "DNS primary/aliases" (run_text (["dscacheutil", "-q", "host", "-a", "name"] | append $device.aliases.0))
  print_step "SSH batch check" (run_text ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", $device.ssh, "true"])
  print_step "TCP/22 check" (run_text ["nc", "-vz", "-G", "3", $device.ip, "22"])
  print_step "UDP/161 check" (run_text ["nc", "-vzu", "-w", "3", $device.ip, "161"])

  if $do_snmpwalk {
    if (can_snmpwalk) {
      print_step "SNMP system walk" (run_text (snmpwalk_cmd $device.ip))
    }
  }

  if (which pup | is-not-empty) {
    print_reachable_metrics $device
  } else {
    print "[SKIP] pup is not installed"
  }
}

def main [
  --target: string = "all",
  --snmpwalk
] {
  let selected = if $target == "all" {
    $devices
  } else {
    $devices | where name == $target
  }

  if ($selected | is-empty) {
    print $"Unknown target: ($target)"
    print $"Known targets: (($devices | get name | str join ', '))"
    exit 2
  }

  print "QNAP SNMP triage probe"
  print "Read-only checks. No secrets are printed by this script except command stderr from local tools."

  if (which pup | is-not-empty) {
    print_auth_status
    print_metric_discovery
    print_metric_tags
  }

  for device in $selected {
    check_device $device $snmpwalk
  }
}
