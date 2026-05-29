local DefaultHost(hostname) = {
  hostname: hostname,
  port: 22,
  actions: {},
};

{
  age_identities: '.age-identities',
  hosts: {
    workstation: DefaultHost('127.0.0.1') {
      actions: {
        copy: { src: 'foobar', dst: 'foobar' },
      },
    },
  },
}
