{
  labels:: error 'labels required',
  name:: error 'name required',
  conf:: error 'conf required',

  deployment: {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name: 'foo',
    },
    spec: {
      template: {
        metadata: {
          labels: $.labels,
        },
        spec: {
          containers_:: {
            foo: {
              image: 'foo/bar:latest',
              resources: {
                limits: self.requests,
                requests: $.conf.Requests,
              },
            },
          },
          containers: local c = self.containers_; [c[n] { name: n } for n in std.objectFields(c)],
        },
      },
    },
  },
}
