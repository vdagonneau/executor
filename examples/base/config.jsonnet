local DefaultHost(hostname) = {
  hostname: hostname,
  port: 22,
};

{
  age_identities: '.age-identities',
  hosts: {
    workstation: DefaultHost('127.0.0.1'),
  },
}
