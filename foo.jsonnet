{
  c:: {
    x: self.y,
    y: 10,
  },
}
+
{
  a: {
    b: $.c.x,
  },
  c+: {
    y: std.trace('', 42),
  },
}
