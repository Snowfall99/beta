all:
	$(MAKE) -C message/
	$(MAKE) -C config/
	$(MAKE) -C cmd/
	${MAKE} -C client/
