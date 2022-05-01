  Listen {
    Subnet:    192.168.0.0/24
    Interface: eth0
    Laddr:     192.168.0.1
  }
  Listen: {
    Subnet:
    Interface:
    Laddr:     10.10.10.1
  }
  Listen: {
    Subnet:
    Interface:
    Laddr: 10.10.20.1
  }
  Subnet {
    Address:   192.168.0.0/24
    RangeFrom: 192.168.0.10
    RangeTo:   192.168.0.99
  }
  Subnet {
    Address:   172.16.0/24
    RangeFrom: 172.16.0.10
    RangeTo:   172.16.0.99
  }

Lease {
  Subnet: 192.168.0.0/24
  MAC: 00:01:02:03:04:05
  IP: 192.168.0.100
}