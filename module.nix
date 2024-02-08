{ lib, config, pkgs, ... }:

with lib;

let
  cfg = config.services.dsmr-exporter;
in {
  options.services.dsmr-exporter = {
    enable = mkEnableOption "DSMR exporter";
    serialDevice = mkOption {
      type = types.str;
    };
    port = mkOption {
      type = types.port;
      default = 9111;
    };
  };

  config = mkIf cfg.enable {
    systemd.services.dsmr-exporter = {
      enable = true;
      description = "DSMR exporter";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];
      serviceConfig = {
        Type = "simple";
        DynamicUser = true;
        ExecStart = "${pkgs.dsmr-exporter}/bin/dsmr-exporter -device ${cfg.serialDevice} -http-addr :${toString cfg.port}";
        Restart = "always";
        RestartSec = 5;

        UMask = "077";
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = false;
        DevicePolicy = "closed";
        DeviceAllow = [cfg.serialDevice];
        SupplementaryGroups = [
          "dialout"
        ];
        PrivateUsers = true;
        ProtectHostname = true;
        ProtectClock = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectKernelLogs = true;
        ProtectControlGroups = true;
        ProtectProc = "invisible";
        ProcSubset = "pid";
        RestrictAddressFamilies = [ "AF_INET" "AF_INET6" ];
        RestrictNamespaces = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        RemoveIPC = true;
        PrivateMounts = true;
        SystemCallArchitectures = "native";
        SystemCallFilter = ["@system-service" "~@privileged" ];
        CapabilityBoundingSet = null;
      };
    };
  };
}
