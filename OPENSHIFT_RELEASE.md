Follow these steps to publish and release the jenkins operator in redhat operator hub:
- Put the right version in OLM_OPERATOR_VERSION in Makefile

```
make olm-image-build 
make olm-image
make olm-publish
make olm-release

```