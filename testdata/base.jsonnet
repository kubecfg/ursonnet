local common = import 'common.libsonnet';
local config = import 'config.libsonnet';

common {
  conf: config,
  name: $.conf.Name,
  labels: {
    app: $.name,
  },
}
