/*jslint browser:true */

var jQuery;
var wssh = {};


(function () {
  // For FormData without getter and setter
  var proto = FormData.prototype,
    data = {};

  if (!proto.get) {
    proto.get = function (name) {
      if (data[name] === undefined) {
        var input = document.querySelector('input[name="' + name + '"]'),
          value;
        if (input) {
          if (input.type === 'file') {
            value = input.files[0];
          } else {
            value = input.value;
          }
          data[name] = value;
        }
      }
      return data[name];
    };
  }

  if (!proto.set) {
    proto.set = function (name, value) {
      data[name] = value;
    };
  }
}());


jQuery(function ($) {
  var status = $('#status'),
    button = $('.btn-primary'),
    form_container = $('.form-container'),
    waiter = $('#waiter'),
    term_type = $('#term'),
    style = {},
    default_title = 'WebSSH',
    title_element = document.querySelector('title'),
    form_id = '#connect',
    debug = window.ssh_server.debug,
    custom_font = document.fonts ? document.fonts.values().next().value : undefined,
    default_fonts,
    DISCONNECTED = 0,
    CONNECTING = 1,
    CONNECTED = 2,
    state = DISCONNECTED,
    messages = { 1: 'This client is connecting ...', 2: 'This client is already connnected.' },
    key_max_size = 16384,
    fields = ['hostname', 'port', 'username'],
    form_keys = fields.concat(['password', 'totp']),
    opts_keys = ['bgcolor', 'title', 'encoding', 'command', 'term', 'fontsize', 'fontcolor', 'cursor'],
    url_form_data = {},
    url_opts_data = {},
    validated_form_data,
    event_origin,
    hostname_tester = /((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))|(^\s*((?=.{1,255}$)(?=.*[A-Za-z].*)[0-9A-Za-z](?:(?:[0-9A-Za-z]|\b-){0,61}[0-9A-Za-z])?(?:\.[0-9A-Za-z](?:(?:[0-9A-Za-z]|\b-){0,61}[0-9A-Za-z])?)*)\s*$)/;

  $("#toolbar").removeClass("visible").addClass("invisible");

  function store_items(names, data) {
    var i, name, value;

    for (i = 0; i < names.length; i++) {
      name = names[i];
      value = data.get(name);
      if (value) {
        window.localStorage.setItem(name, value);
      }
    }
  }


  function restore_items(names) {
    var i, name, value;

    for (i = 0; i < names.length; i++) {
      name = names[i];
      value = window.localStorage.getItem(name);
      if (value) {
        $('#' + name).val(value);
      }
    }
  }


  function populate_form(data) {
    var names = form_keys.concat(['passphrase']),
      i, name;

    for (i = 0; i < names.length; i++) {
      name = names[i];
      $('#' + name).val(data.get(name));
    }
  }


  function get_object_length(object) {
    return Object.keys(object).length;
  }


  function decode_uri_component(uri) {
    try {
      return decodeURIComponent(uri);
    } catch (e) {
      console.error(e);
    }
    return '';
  }


  function decode_password(encoded) {
    try {
      // see: https://developer.mozilla.org/en-US/docs/Web/API/Window/atob
      return window.atob(encoded);
    } catch (e) {
      console.error(e);
    }
    return null;
  }


  function parse_url_data(string, form_keys, opts_keys, form_map, opts_map) {
    var i, pair, key, val,
      arr = string.split('&');

    const ssh_server = window.ssh_server;

    // set values from SSH Server if `SSH_AUTO_LOGIN` is enabled.
    if (ssh_server.auto_login) {
      form_map['hostname'] = ssh_server.host;
      form_map['port'] = ssh_server.port;
      form_map['username'] = ssh_server.user;
      form_map['password'] = ssh_server.pass;
    }

    // continue with URL query params to allow client overrides.
    for (i = 0; i < arr.length; i++) {
      pair = arr[i].split('=');
      key = pair[0].trim().toLowerCase();
      val = pair.slice(1).join('=').trim();

      if (form_keys.indexOf(key) >= 0) {
        form_map[key] = val;
      } else if (opts_keys.indexOf(key) >= 0) {
        opts_map[key] = val;
      }
    }

    // if SSH Server provides a password, then skip B64 decode.
    if (!ssh_server.auto_login && form_map.password) {
      form_map.password = decode_password(form_map.password);
    }
  }

  function parse_xterm_style() {
    var text = $('.xterm-helpers style').text();
    var arr = text.split('xterm-normal-char{width:');
    style.width = parseFloat(arr[1]);
    arr = text.split('div{height:');
    style.height = parseFloat(arr[1]);
  }


  function get_cell_size(term) {
    style.width = term._core._renderService._renderer.dimensions.actualCellWidth;
    style.height = term._core._renderService._renderer.dimensions.actualCellHeight;
  }

  function toggle_fullscreen(term) {
    $('#terminal .terminal').toggleClass('fullscreen');
    term.fitAddon.fit();
  }

  function current_geometry(term) {
    if (!style.width || !style.height) {
      try {
        get_cell_size(term);
      } catch (TypeError) {
        parse_xterm_style();
      }
    }

    var cols = parseInt(window.innerWidth / style.width, 10) - 1;
    var rows = parseInt(window.innerHeight / style.height, 10);
    return { 'cols': cols, 'rows': rows };
  }


  function resize_terminal(term) {
    var geometry = current_geometry(term);
    term.on_resize(geometry.cols, geometry.rows);
  }


  function set_backgound_color(term, color) {
    term.setOption('theme', {
      background: color
    });
  }

  function set_font_color(term, color) {
    term.setOption('theme', {
      foreground: color
    });
  }

  function custom_font_is_loaded() {
    if (!custom_font) {
      console.log('No custom font specified.');
    } else {
      console.log('Status of custom font ' + custom_font.family + ': ' + custom_font.status);
      if (custom_font.status === 'loaded') {
        return true;
      }
      if (custom_font.status === 'unloaded') {
        return false;
      }
    }
  }

  function update_font_family(term) {
    if (term.font_family_updated) {
      console.log('Already using custom font family');
      return;
    }

    if (!default_fonts) {
      default_fonts = term.getOption('fontFamily');
    }

    if (custom_font_is_loaded()) {
      var new_fonts = custom_font.family + ', ' + default_fonts;
      term.setOption('fontFamily', new_fonts);
      term.font_family_updated = true;
      console.log('Using custom font family ' + new_fonts);
    }
  }

  function reset_font_family(term) {
    if (!term.font_family_updated) {
      console.log('Already using default font family');
      return;
    }

    if (default_fonts) {
      term.setOption('fontFamily', default_fonts);
      term.font_family_updated = false;
      console.log('Using default font family ' + default_fonts);
    }
  }

  function format_geometry(cols, rows) {
    return JSON.stringify({ 'cols': cols, 'rows': rows });
  }

  function read_as_text_with_decoder(file, callback, decoder) {
    var reader = new window.FileReader();

    if (decoder === undefined) {
      decoder = new window.TextDecoder('utf-8', { 'fatal': true });
    }

    reader.onload = function () {
      var text;
      try {
        text = decoder.decode(reader.result);
      } catch (TypeError) {
        console.log('Decoding error happened.');
      } finally {
        if (callback) {
          callback(text);
        }
      }
    };

    reader.onerror = function (e) {
      console.error(e);
    };

    reader.readAsArrayBuffer(file);
  }

  function read_as_text_with_encoding(file, callback, encoding) {
    var reader = new window.FileReader();

    if (encoding === undefined) {
      encoding = 'utf-8';
    }

    reader.onload = function () {
      if (callback) {
        callback(reader.result);
      }
    };

    reader.onerror = function (e) {
      console.error(e);
    };

    reader.readAsText(file, encoding);
  }

  function read_file_as_text(file, callback, decoder) {
    if (!window.TextDecoder) {
      read_as_text_with_encoding(file, callback, decoder);
    } else {
      read_as_text_with_decoder(file, callback, decoder);
    }
  }

  function reset_wssh() {
    var name;

    for (name in wssh) {
      if (wssh.hasOwnProperty(name) && name !== 'connect') {
        delete wssh[name];
      }
    }
  }

  function log_status(text, to_populate) {
    console.log(text);
    status.html(text.split('\n').join('<br/>'));

    if (to_populate && validated_form_data) {
      populate_form(validated_form_data);
      validated_form_data = undefined;
    }

    if (waiter.css('display') !== 'none') {
      waiter.hide();
    }

    if (form_container.css('display') === 'none') {
      form_container.show();
    }
  }

  function ajax_complete_callback(resp) {
    button.prop('disabled', false);

    if (resp.status !== 200) {
      log_status(resp.status + ': ' + resp.statusText, true);
      state = DISCONNECTED;
      return;
    }

    var msg = resp.responseJSON;
    if (!msg.id) {
      log_status(msg.status, true);
      state = DISCONNECTED;
      return;
    }

    var ws_url = window.location.href.split(/\?|#/, 1)[0].replace('http', 'ws'),
      join = (ws_url[ws_url.length - 1] === '/' ? '' : '/'),
      url = ws_url + join + 'ws?id=' + msg.id,
      sock = new window.WebSocket(url),
      encoding = 'utf-8',
      decoder = window.TextDecoder ? new window.TextDecoder(encoding) : encoding,
      terminal = document.getElementById('terminal'),
      termOptions = {
        cursorBlink: true,
        theme: {
          background: url_opts_data.bgcolor || 'black',
          foreground: url_opts_data.fontcolor || 'white',
          cursor: url_opts_data.cursor || url_opts_data.fontcolor || 'white'
        }
      };

    if (url_opts_data.fontsize) {
      var fontsize = window.parseInt(url_opts_data.fontsize);
      if (fontsize && fontsize > 0) {
        termOptions.fontSize = fontsize;
      }
    }

    var term = new window.Terminal(termOptions);

    term.fitAddon = new window.FitAddon.FitAddon();
    term.loadAddon(term.fitAddon);

    console.log(url);
    if (!msg.encoding) {
      console.log('Unable to detect the default encoding of your server');
      msg.encoding = encoding;
    } else {
      console.log('The deault encoding of your server is ' + msg.encoding);
    }

    const commandCatalogEntryTemplate = `
<a href="#" class="list-group-item list-group-item-action cmd-exec" data-cmd="{{key}}">
  <div class="d-flex w-100 justify-content-between">
    <h5 class="mb-1"><code>{{name}}</code></h5>
    <small>
      {{#each tags}}
      <span class="badge text-bg-light rounded-pill">{{.}}</span>
      {{/each}}
    </small>
  </div>
  <p class="mb-1">{{desc}}</p>
  <small>
    <button type="button" class="btn btn-link btn-sm cmd-link" data-cmd="{{key}}" data-cmd-link="man">learn more about <code>{{name}}</code></button>
  </small>
</a>`;
    const commandArgumentInputTemplate = `
<div class="input-group flex-nowrap" data-cmd="{{cmd.key}}" data-cmd-arg="{{key}}">
  <span class="input-group-text" id="addon-wrapping">{{label}}</span>
  <div class="form-floating">
    <input id="cmd-arg-{{key}}" data-cmd="{{cmd.key}}" data-arg="{{key}}" type="text" class="form-control cmd-arg" placeholder="{{desc}}">
    <label for="cmd-arg-{{key}}">{{desc}}</label>
  </div>
</div>`;

    const $toolbar = $("#toolbar");
    const $commandsButton = $toolbar.find("#commandsButton");
    const $disconnectButton = $toolbar.find("#disconnectButton");
    const $transcriptButton = $toolbar.find("#transcriptButton");
    const $cloudRunButton = $toolbar.find("#cloudRunButton");
    const $downloadButton = $toolbar.find("#downloadButton");
    const $editorButton = $toolbar.find("#editorButton");

    const $buttons = $toolbar
      .add($cloudRunButton)
      .add($downloadButton)
      .add($transcriptButton)
      .add($commandsButton)
      .add($editorButton)
      .add($disconnectButton);

    const transcriptModalElement = document.getElementById("transcriptModal");
    const $transcriptModal = $(transcriptModalElement);
    const transcriptModal = new bootstrap.Modal(transcriptModalElement, {
      keyboard: true, focus: false, backdrop: true,
    });
    const $copyTranscriptButton = $transcriptModal.find("#copyTranscriptBtn");
    const $clearTranscriptButton = $transcriptModal.find("#clearTranscriptBtn");
    const $transcriptContent = $transcriptModal.find("#transcript");
    const $isTranscriptEnabled = $transcriptModal.find("#isTranscriptEnabled");
    let isTranscriptEnabled = false;
    const transcript = []

    const offcanvasElement = document.getElementById("offcanvas");
    const $offcanvas = $(offcanvasElement);
    const offcanvas = new bootstrap.Offcanvas(offcanvasElement);
    const $commandsCatalog = $offcanvas.find("#commandsCatalog");

    const commandConfigModalElement = document.getElementById("commandConfigModal");
    const commandConfigModal = new bootstrap.Modal(commandConfigModalElement, {
      keyboard: false, focus: true, backdrop: 'static',
    });
    const $commandConfigModal = $(commandConfigModalElement);
    const $command = $commandConfigModal.find("#command");
    const $commandPreview = $commandConfigModal.find("#commandPreview");
    const $commandConfig = $commandConfigModal.find("#commandConfig");
    const $runCommandButton = $commandConfigModal.find("#runCommandBtn");
    const $cancelCommandButton = $commandConfigModal.find("#cancelCommandBtn");
    const commandQueue = [];

    const downloadModalElement = document.getElementById("downloadModal");
    const downloadModal = new bootstrap.Modal(downloadModalElement, {
      keyboard: true, focus: true, backdrop: true,
    });
    const $downloadModal = $(downloadModalElement)
    const $downloadFileButton = $downloadModal.find("#downloadBtn");
    const $downloadFile = $downloadModal.find("#downloadFile");

    // const controlSequenceRegex = /\x1B\[([0-9]*;)*[\?]?[0-9]*[a-zA-Z]/g;
    const controlSequenceRegex = /\x1B(([\[\]]([0-9]*;)*[\?>]?([0-9]*[a-zA-Z])?)|([\(\)>=][0A-Z]?))?/g;
    const fixResizeRegex = /[\r\n]{2}(.+@?.+?:.+?[#\$]?\s)/g;

    const getTranscript = _.bind(function () {
      const {
        transcript,
        fixResizeRegex,
      } = this;

      if (_.isEmpty(transcript)) {
        return null;
      }

      let text = _.join(transcript, '');
      const rgx = /.\x08/;
      while (text.indexOf("\b") != -1) {
        text = text.replace(rgx, "");
      }
      text = text.replaceAll(fixResizeRegex, "\n$1\r")

      if (_.isEmpty(text)) {
        return null;
      }
      return text;
    }, { term, transcript, fixResizeRegex });

    window.commandQueue = commandQueue;
    window.transcript = transcript;
    window.getTranscript = getTranscript;
    window.logTranscript = _.bind(function () {
      const { getTranscript } = this;
      console.log(getTranscript());
    }, { getTranscript });

    const addToTranscript = (function (
      transcript,
      timeout = 1000
    ) {
      const data = [];

      let timer;

      return (text) => {
        clearTimeout(timer);
        if (isTranscriptEnabled) {
          data.push(text);
        }
        timer = setTimeout(() => {
          if (data.length > 0) {
            let txt = data.join("");
            txt = txt.replaceAll(controlSequenceRegex, "");
            txt = _.join(_.compact(_.split(txt, '\r\n')), '\n');
            if (txt) {
              transcript.push(txt);
            }
            data.length = 0;
          }
        }, timeout);
      };
    })(transcript, 2000);

    const copyTranscriptButton = new ClipboardJS(
      $copyTranscriptButton.get(0), {
      text: getTranscript,
    });

    const $writeCallbacks = $.Callbacks();
    const $commandCallbacks = $.Callbacks();

    transcriptModalElement
      .addEventListener('show.bs.modal',
        $.proxy(function () {
          const {
            getTranscript,
            $transcriptContent,
          } = this;
          $transcriptContent.text(
            _.defaultTo(getTranscript(), "EMPTY...")
          );
        }, { getTranscript, $transcriptContent }));

    $clearTranscriptButton.on("click", {
      term, transcript, transcriptModal,
    }, function (e) {
      const data = e.data;
      data.transcript.length = 0;
      transcriptModal.hide();
    });

    $isTranscriptEnabled.on("change", {
      term, transcript, transcriptModal,
    }, function () {
      isTranscriptEnabled = this.checked;
    });

    $disconnectButton.on("click",
      { wssh, sock, term },
      function (e) {
        const { sock } = e.data;
        sock.send(JSON.stringify({
          'data': "\n\rexit\n\r"
        }));
        sock.close(1000, "exit");
      });

    copyTranscriptButton.on('success',
      function (e) {
        transcriptModal.hide();
        e.clearSelection();
      });

    $commandCallbacks.add(
      _.bind(function (op, cmd) {
        const {
          commandConfigModal,
          $command,
          $commandPreview,
          $commandConfig,
        } = this;

        console.log(`${op} => ${cmd.name}`, cmd);

        if (op == "update") {
          return false;
        }

        commandConfigModal.hide();
        $commandConfig.empty();
        $command.text("command");
        $commandPreview.addClass('d-none').empty();

        return true;
      }, {
        commandConfigModal,
        $command,
        $commandPreview,
        $commandConfig
      }));

    const getArgs = function(cmd) {
      const $args = $commandConfig.find(".cmd-arg");
      const args = {};
      $args.each(function () {
        const $arg = $(this);
        const cmdKey = $arg.data("cmd");
        const argKey = $arg.data("arg");
        if (_.eq(cmd.key, cmdKey)) {
          const argPath = ["args", argKey];
          const arg = _.get(cmd, argPath);
          const val = $arg.val();
          if (arg && val) {
            _.set(args, argKey, val);
          }
        }
      });
      return args;
    };

    $commandConfigModal.on("keyup.cmd-arg",
      _.debounce(function () {
        const cmd = commandQueue[0];
        const args = getArgs(cmd);
        if ($.isEmptyObject(args)) {
          $commandPreview.addClass('d-none').empty();
        } else {
          $commandPreview
            .text(cmd.providers.cmd(args))
            .removeClass('d-none');
        }
        $commandCallbacks.fire("update", cmd);
      }, 300));

    $runCommandButton.on("click", function () {
      const cmd = commandQueue.shift();
      const args = getArgs(cmd);
      if (!$.isEmptyObject(args)) {
        wssh.send(cmd.providers.cmd(args) + "\n");
      }
      $commandCallbacks.fire("exec", cmd);
    });

    $cancelCommandButton.on("click", function () {
      const cmd = commandQueue.shift();
      $commandCallbacks.fire("abort", cmd);
    });

    $downloadFileButton.on("click", function () {
      const relativePath = $downloadFile.val();
      if (relativePath) {
        const path = _.join(["/dl", relativePath], "/");
        window.open(path, "_blank");
      }
      downloadModal.hide();
    });

    downloadModalElement
      .addEventListener('hide.bs.modal',
        $.proxy(function () {
          const { $downloadFile } = this;
          $downloadFile.val("");
        }, { $downloadFile }));

    $editorButton.on("click", {
      ssh_server: window.ssh_server,
    }, function (e) {
      const { ssh_server } = e.data;
      if (_.eq(ssh_server.flavor, "dev")) {
        window.open("/dev/", "_blank");
      }
    });

    const onCatalogsLoaded = function (commands) {
      htmlTemplate = Handlebars.compile(commandCatalogEntryTemplate);
      argTemplate = Handlebars.compile(commandArgumentInputTemplate);

      $commandsCatalog.empty();

      $.each(commands.cmds, function (key, cmd) {
        cmd.key = key;
        cmd.name = _.defaultTo(cmd.name, key);
        cmd.providers = {
          cmd: Handlebars.compile(cmd.tpl),
        };
        cmd = _.omit(cmd, 'tpl');
        $commandsCatalog.append(htmlTemplate(cmd));
      });

      $commandsCatalog.off("click.cmd-exec", ".cmd-exec");
      $commandsCatalog.on("click.cmd-exec",
        ".cmd-exec", commands,
        function (e) {
          const { cmds } = e.data;
          const $el = $(this);
          const cmdKey = $el.data("cmd");
          const cmd = _.get(cmds, cmdKey);

          $commandConfig.empty();
          $.each(cmd.args, function (key, arg) {
            const _cmd = _.extend({}, cmd);
            const _arg = _.extend({}, arg);
            _arg.key = key;
            _arg.label = _.defaultTo(_arg.label, key);
            _arg.cmd = _.omit(_cmd, ['args', 'providers']);
            $commandConfig.append(argTemplate(_arg));
          });
          commandQueue.push(cmd);
          $command.empty().text(cmd.name);
          offcanvas.hide();
          commandConfigModal.show();
        });

      $commandsCatalog.off("click.cmd-link", ".cmd-link");
      $commandsCatalog.on("click.cmd-link",
        ".cmd-link", commands,
        function (e) {
          const { cmds } = e.data;
          const $el = $(this);
          const key = $el.data("cmd");
          const link = $el.data("cmdLink");
          const cmd = _.get(cmds, key);
          const linkPath = ["links", link];
          if (_.has(cmd, linkPath)) {
            const linkURL = _.get(cmd, linkPath);
            window.open(linkURL, "_blank");
            e.stopImmediatePropagation();
          }
        });
    };

    const initialize = function () {
      // load all required catalogs
      $.when(
        $.getJSON("/static/json/commands.json")
      ).done(onCatalogsLoaded);
      $buttons.removeClass("invisible").addClass("visible");
      transcript.length = 0
    }

    const term_write = function (text) {
      if (term) {
        if (text) {
          term.write(text);
        }
        if (!term.resized) {
          resize_terminal(term);
          term.resized = true;
        }
      }
    };

    $writeCallbacks.add(term_write);
    $writeCallbacks.add(addToTranscript);

    function set_encoding(new_encoding) {
      // for console use
      if (!new_encoding) {
        console.log('An encoding is required');
        return;
      }

      if (!window.TextDecoder) {
        decoder = new_encoding;
        encoding = decoder;
        console.log('Set encoding to ' + encoding);
      } else {
        try {
          decoder = new window.TextDecoder(new_encoding);
          encoding = decoder.encoding;
          console.log('Set encoding to ' + encoding);
        } catch (RangeError) {
          console.log('Unknown encoding ' + new_encoding);
          return false;
        }
      }
    }

    wssh.set_encoding = set_encoding;

    if (url_opts_data.encoding) {
      if (set_encoding(url_opts_data.encoding) === false) {
        set_encoding(msg.encoding);
      }
    } else {
      set_encoding(msg.encoding);
    }

    wssh.geometry = function () {
      // for console use
      var geometry = current_geometry(term);
      console.log('Current window geometry: ' + JSON.stringify(geometry));
    };

    wssh.send = function (data) {
      // for console use
      if (!sock) {
        console.log('Websocket was already closed');
        return;
      }

      if (typeof data !== 'string') {
        console.log('Only string is allowed');
        return;
      }

      try {
        JSON.parse(data)/* ; */
        sock.send(data);
      } catch (SyntaxError) {
        data = data.trim() + '\r';
        sock.send(JSON.stringify({ 'data': data }));
      }
    };

    wssh.reset_encoding = function () {
      // for console use
      if (encoding === msg.encoding) {
        console.log('Already reset to ' + msg.encoding);
      } else {
        set_encoding(msg.encoding);
      }
    };

    wssh.resize = function (cols, rows) {
      // for console use
      if (term === undefined) {
        console.log('Terminal was already destroryed');
        return;
      }

      var valid_args = false;

      if (cols > 0 && rows > 0) {
        var geometry = current_geometry(term);
        if (cols <= geometry.cols && rows <= geometry.rows) {
          valid_args = true;
        }
      }

      if (!valid_args) {
        console.log('Unable to resize terminal to geometry: ' + format_geometry(cols, rows));
      } else {
        term.on_resize(cols, rows);
      }
    };

    wssh.set_bgcolor = function (color) {
      set_backgound_color(term, color);
    };

    wssh.set_fontcolor = function (color) {
      set_font_color(term, color);
    };

    wssh.custom_font = function () {
      update_font_family(term);
    };

    wssh.default_font = function () {
      reset_font_family(term);
    };

    term.on_resize = function (cols, rows) {
      if (cols !== this.cols || rows !== this.rows) {
        console.log('Resizing terminal to geometry: ' + format_geometry(cols, rows));
        this.resize(cols, rows);
        sock.send(JSON.stringify({ 'resize': [cols, rows] }));
      }
    };

    function termInputProxy(fn) {
      return (input) => {
        // console.log("data:", data);
        fn.call(this, input);
      };
    }

    term.onData(termInputProxy(function (data) {
      sock.send(JSON.stringify({ 'data': data }));
    }));

    sock.onopen = function () {
      term.open(terminal);
      toggle_fullscreen(term);
      update_font_family(term);
      term.focus();
      state = CONNECTED;
      title_element.text = url_opts_data.title || default_title;
      if (url_opts_data.command) {
        setTimeout(function () {
          sock.send(JSON.stringify({ 'data': url_opts_data.command + '\r' }));
        }, 500);
      }
      initialize();
    };

    sock.onmessage = function (msg) {
      read_file_as_text(msg.data, $writeCallbacks.fire, decoder);
    };

    sock.onerror = function (e) {
      console.error(e);
    };

    sock.onclose = function (e) {
      term.dispose();
      term = undefined;
      sock = undefined;
      reset_wssh();
      log_status(e.reason, true);
      state = DISCONNECTED;
      default_title = 'Cloud Run SSH server';
      title_element.text = default_title;
      $buttons.removeClass("visible").addClass("invisible");
      transcript.length = 0
      offcanvas.hide();
      downloadModal.hide();
      transcriptModal.hide();
      commandConfigModal.hide();
    };

    $(window).resize(function () {
      if (term) {
        resize_terminal(term);
      }
    });
  }


  function wrap_object(opts) {
    var obj = {};

    obj.get = function (attr) {
      return opts[attr] || '';
    };

    obj.set = function (attr, val) {
      opts[attr] = val;
    };

    return obj;
  }


  function clean_data(data) {
    var i, attr, val;
    var attrs = form_keys.concat(['privatekey', 'passphrase']);

    for (i = 0; i < attrs.length; i++) {
      attr = attrs[i];
      val = data.get(attr);
      if (typeof val === 'string') {
        data.set(attr, val.trim());
      }
    }
  }

  function validate_form_data(data) {
    clean_data(data);

    var hostname = data.get('hostname'),
      port = data.get('port'),
      username = data.get('username'),
      pk = data.get('privatekey'),
      result = {
        valid: false,
        data: data,
        title: ''
      },
      errors = [], size;

    if (!hostname) {
      errors.push('Value of hostname is required.');
    } else {
      if (!hostname_tester.test(hostname)) {
        errors.push('Invalid hostname: ' + hostname);
      }
    }

    if (!port) {
      port = 2222;
    } else {
      if (!(port > 0 && port <= 65535)) {
        errors.push('Invalid port: ' + port);
      }
    }

    if (!username) {
      errors.push('Value of username is required.');
    }

    if (pk) {
      size = pk.size || pk.length;
      if (size > key_max_size) {
        errors.push('Invalid private key: ' + pk.name || '');
      }
    }

    if (!errors.length || debug) {
      result.valid = true;
      result.title = username + '@' + hostname + ':' + port;
    }
    result.errors = errors;

    return result;
  }

  // Fix empty input file ajax submission error for safari 11.x
  function disable_file_inputs(inputs) {
    var i, input;

    for (i = 0; i < inputs.length; i++) {
      input = inputs[i];
      if (input.files.length === 0) {
        input.setAttribute('disabled', '');
      }
    }
  }

  function enable_file_inputs(inputs) {
    var i;

    for (i = 0; i < inputs.length; i++) {
      inputs[i].removeAttribute('disabled');
    }
  }

  function connect_without_options() {
    // use data from the form
    var form = document.querySelector(form_id),
      inputs = form.querySelectorAll('input[type="file"]'),
      url = form.action,
      data, pk;

    disable_file_inputs(inputs);
    data = new FormData(form);
    pk = data.get('privatekey');
    enable_file_inputs(inputs);

    function ajax_post() {
      status.text('');
      button.prop('disabled', true);

      $.ajax({
        url: url,
        type: 'post',
        data: data,
        complete: ajax_complete_callback,
        cache: false,
        contentType: false,
        processData: false
      });
    }

    var result = validate_form_data(data);
    if (!result.valid) {
      log_status(result.errors.join('\n'));
      return;
    }

    if (pk && pk.size && !debug) {
      read_file_as_text(pk, function (text) {
        if (text === undefined) {
          log_status('Invalid private key: ' + pk.name);
        } else {
          ajax_post();
        }
      });
    } else {
      ajax_post();
    }

    return result;
  }

  function connect_with_options(data) {
    // use data from the arguments
    var form = document.querySelector(form_id),
      url = data.url || form.action,
      _xsrf = form.querySelector('input[name="_xsrf"]');

    var result = validate_form_data(wrap_object(data));
    if (!result.valid) {
      log_status(result.errors.join('\n'));
      return;
    }

    data.term = term_type.val();
    data._xsrf = _xsrf.value;
    if (event_origin) {
      data._origin = event_origin;
    }

    status.text('');
    button.prop('disabled', true);

    $.ajax({
      url: url,
      type: 'post',
      data: data,
      complete: ajax_complete_callback
    });

    return result;
  }

  function connect(hostname, port, username, password, privatekey, passphrase, totp) {
    // for console use
    var result, opts;

    if (state !== DISCONNECTED) {
      console.log(messages[state]);
      return;
    }

    if (hostname === undefined) {
      result = connect_without_options();
    } else {
      if (typeof hostname === 'string') {
        opts = {
          hostname: hostname,
          port: port,
          username: username,
          password: password,
          privatekey: privatekey,
          passphrase: passphrase,
          totp: totp
        };
      } else {
        opts = hostname;
      }

      result = connect_with_options(opts);
    }

    if (result) {
      state = CONNECTING;
      default_title = result.title;
      if (hostname) {
        validated_form_data = result.data;
      }
      store_items(fields, result.data);
    }
  }

  wssh.connect = connect;

  $(form_id).submit(function (event) {
    event.preventDefault();
    connect();
  });

  function cross_origin_connect(event) {
    console.log(event.origin);
    var prop = 'connect',
      args;

    try {
      args = JSON.parse(event.data);
    } catch (SyntaxError) {
      args = event.data.split('|');
    }

    if (!Array.isArray(args)) {
      args = [args];
    }

    try {
      event_origin = event.origin;
      wssh[prop].apply(wssh, args);
    } finally {
      event_origin = undefined;
    }
  }

  window.addEventListener('message', cross_origin_connect, false);

  if (document.fonts) {
    document.fonts.ready.then(
      function () {
        if (custom_font_is_loaded() === false) {
          document.body.style.fontFamily = custom_font.family;
        }
      }
    );
  }

  parse_url_data(
    decode_uri_component(window.location.search.substring(1)) + '&' + decode_uri_component(window.location.hash.substring(1)),
    form_keys, opts_keys, url_form_data, url_opts_data
  );
  // console.log(url_form_data);
  // console.log(url_opts_data);

  if (url_opts_data.term) {
    term_type.val(url_opts_data.term);
  }

  if (url_form_data.password === null) {
    log_status('Password via url must be encoded in base64.');
  } else {
    if (get_object_length(url_form_data)) {
      waiter.show();
      connect(url_form_data);
    } else {
      restore_items(fields);
      form_container.show();
    }
  }

});
