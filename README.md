<p align="center">
  <img height="300" src="https://sr.ht/2mc0.png">
</p>

tor-controller
==============

[![Build Status](https://img.shields.io/travis-ci/kragniz/tor-controller.svg?style=flat-square)](https://travis-ci.org/kragniz/tor-controller)

Sprinkle some onions on your kubernetes clusters.

Quickstart
----------

Put your private key into a secret:

    kubectl create secret generic bmy7nlgozpyn26tv --from-file=private_key

Create an onion service:

```yaml
apiVersion: tor.k8s.io/v1alpha1
kind: OnionService
metadata:
  name: example-onion
spec:
  selector:
    app: httpd
  ports:
    - targetPort: 8080
      publicPort: 80
  privateKeySecret:
    name: bmy7nlgozpyn26tv
    key: private_key
```

This creates:

- a service, which is used to send traffic to application pods
- a configmap containing tor configuration pointing at the service
- tor daemon pod, which serves incoming traffic from the tor network

<p align="center">
  <img src="https://sr.ht/6WbX.png">
</p>
