all:
	$(MAKE) -C common/
	$(MAKE) -C config/
	$(MAKE) -C cmd/
	${MAKE} -C client/
