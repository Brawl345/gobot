{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.gobot;
  defaultUser = "gobot";
  inherit (lib)
    mkEnableOption
    mkMerge
    mkPackageOption
    mkOption
    mkIf
    types
    optionalAttrs
    optional
    optionalString
    ;
in
{
  options.services.gobot = {
    enable = mkEnableOption "Gobot Telegram bot";

    package = mkPackageOption pkgs "gobot" { };

    user = mkOption {
      type = types.str;
      default = defaultUser;
      description = "User under which Gobot runs.";
    };

    adminId = mkOption {
      type = types.int;
      description = "Admin ID";
    };

    botTokenFile = mkOption {
      type = types.path;
      description = "File containing Telegram Bot Token";
    };

    database = {
      host = lib.mkOption {
        type = types.str;
        description = "Database host.";
        default = "localhost";
      };

      port = mkOption {
        type = types.port;
        default = 3306;
        description = "Database port";
      };

      name = lib.mkOption {
        type = types.str;
        description = "Database name.";
        default = "gobot";
      };

      user = lib.mkOption {
        type = types.str;
        description = "Database username.";
        default = "gobot";
      };

      passwordFile = lib.mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "Database user password file.";
      };

      socket = mkOption {
        type = types.nullOr types.path;
        default = "/run/mysqld/mysqld.sock";
        example = "/run/mysqld/mysqld.sock";
        description = "Path to the unix socket file to use for authentication.";
      };

      createLocally = mkOption {
        type = types.bool;
        default = true;
        description = "Create the database locally";
      };
    };

    printMsgs = mkOption {
      type = types.bool;
      default = false;
      description = "Print all messages the bot receives";
    };

    prettyPrintLog = mkOption {
      type = types.bool;
      default = false;
      description = "Pretty print logs";
    };

    debug = mkOption {
      type = types.bool;
      default = false;
      description = "Enable debug mode";
    };

    useWebhook = mkOption {
      type = types.bool;
      default = false;
      description = "Use webhook instead of long polling";
    };

    port = mkOption {
      type = types.port;
      default = 8080;
      description = "Port for the webhook server";
    };

    webhookPublicUrl = mkOption {
      type = types.str;
      default = "";
      description = "Public URL for webhook";
    };

    webhookUrlPath = mkOption {
      type = types.strMatching "\/.+";
      default = "/webhook";
      description = "Custom path for the webhook";
    };

    webhookSecret = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "Secret for the webhook";
    };

    webhookSecretFile = mkOption {
      type = types.nullOr types.path;
      default = null;
      description = "Path to a file containing the secret for the webhook";
    };
  };

  config = mkIf cfg.enable {

    assertions = [
      {
        assertion = !(cfg.webhookSecret != null && cfg.webhookSecretFile != null);
        message = "Only one of webhookSecret or webhookSecretFile can be set.";
      }
      {
        assertion = !(cfg.database.socket != null && cfg.database.passwordFile != null);
        message = "Only one of services.gobot.database.socket or services.gobot.database.passwordFile can be set.";
      }
      {
        assertion = cfg.database.socket == null && cfg.database.passwordFile == null;
        message = "Either services.gobot.database.socket or services.gobot.database.passwordFile must be set.";
      }
    ];

    services.mysql = lib.mkIf cfg.database.createLocally {
      enable = lib.mkDefault true;
      package = lib.mkDefault pkgs.mariadb;
      ensureDatabases = [ cfg.database.name ];
      ensureUsers = [
        {
          name = cfg.database.user;
          ensurePermissions = {
            "${cfg.database.name}.*" = "ALL PRIVILEGES";
          };
        }
      ];
    };

    systemd.services.gobot = {
      description = "Gobot Telegram Bot";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];

      script = ''
        export BOT_TOKEN="$(< $CREDENTIALS_DIRECTORY/BOT_TOKEN )"
        ${optionalString (cfg.database.passwordFile != null) ''
          export MYSQL_PASSWORD="$(< $CREDENTIALS_DIRECTORY/MYSQL_PASSWORD )"
        ''}
        ${optionalString (cfg.useWebhook && cfg.webhookSecretFile != null) ''
          export WEBHOOK_SECRET="$(< $CREDENTIALS_DIRECTORY/WEBHOOK_SECRET )"
        ''}

        exec ${cfg.package}/bin/gobot
      '';

      serviceConfig = {
        LoadCredential =
          [
            "BOT_TOKEN:${cfg.botTokenFile}"
          ]
          ++ optional (
            cfg.useWebhook && cfg.webhookSecretFile != null
          ) "WEBHOOK_SECRET:${cfg.webhookSecretFile}"
          ++ optional (cfg.database.passwordFile != null) "MYSQL_PASSWORD:${cfg.database.passwordFile}";

        Restart = "always";
        User = cfg.user;
        Group = defaultUser;
      };

      environment = mkMerge [
        {
          ADMIN_ID = toString cfg.adminId;
          MYSQL_HOST = cfg.database.host;
          MYSQL_PORT = toString cfg.database.port;
          MYSQL_USER = cfg.database.user;
          MYSQL_DB = cfg.database.name;
          MYSQL_SOCKET = cfg.database.socket;
        }
        (mkIf cfg.useWebhook {
          PORT = toString cfg.port;
          WEBHOOK_PUBLIC_URL = cfg.webhookPublicUrl;
          WEBHOOK_URL_PATH = cfg.webhookUrlPath;
          WEBHOOK_SECRET = optionalString (cfg.webhookSecret != null) cfg.webhookSecret;
        })
        (mkIf cfg.printMsgs {
          PRINT_MSGS = "true";
        })
        (mkIf cfg.debug {
          DEBUG = "true";
        })
        (mkIf cfg.prettyPrintLog {
          PRETTY_PRINT_LOG = "true";
        })
      ];
    };

    users = optionalAttrs (cfg.user == defaultUser) {
      users.${defaultUser} = {
        isSystemUser = true;
        group = defaultUser;
        description = "Gobot Telegram bot user";
      };

      groups.${defaultUser} = { };
    };

  };

}
