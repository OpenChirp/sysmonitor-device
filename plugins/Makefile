# This should be a list of all plugin names, such that the plugin is simply one file <plugin_name>.go
PLUGINS=raspberrypi network

# Set your platform's dynamic library's file extension
PLUGIN_EXT=.so

.PHONY: all clean

TARGETS=$(addsuffix $(PLUGIN_EXT),$(PLUGINS))

all: $(TARGETS)

%$(PLUGIN_EXT): %/*.go
	go build -buildmode=plugin $^

clean:
	$(RM) $(TARGETS)
