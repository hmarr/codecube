var aceMode = function(lang) {
  if (lang === "c") {
    return "c_cpp";
  }
  return lang;
}

var getId = function() {
  return window.location.pathname.replace(/\/$/, '').split('/').pop();
}

var generateId = function() {
  var chars = [];
  var choices = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

  for (var i = 0; i < 6; i++) {
    chars.push(choices.charAt(Math.floor(Math.random() * choices.length)));
  }
  return chars.join('');
}

$(function() {
  var editor = ace.edit("editor");
  editor.setTheme("ace/theme/monokai");

  var langBox = $('#language');
  var lang = langBox.val();
  var running = false;

  var updateLanguage = function() {
    lang = langBox.val();
    editor.getSession().setMode("ace/mode/" + aceMode(lang));
  }

  var output = [];

  var id = getId();

  if (id == 'demo') {
    editor.setValue([
      '5.times do',
      '  puts "Hello, world!"',
      '  sleep 0.5',
      'end'
    ].join('\n'));
    langBox.val('ruby');
    editor.getSelection().clearSelection();
    updateLanguage();
    id = "";
  }

  var regenerateId = function() {
    id = generateId();
    history.replaceState(null, null, '/' + id);
    owner = true;
  };

  if (typeof id === undefined || id == "") {
    regenerateId();
  }

  var evtSource;
  var reopenEventSource = function() {
    if (evtSource) {
      evtSource.close();
    }

    evtSource = new EventSource("/api/event-stream/?id=" + id);
    evtSource.onmessage = function(e) {
      output.push(e.data);
      $('#console').text(output.join('\n'));
    }
  };
  reopenEventSource();

  var owner = true;

  $.post('/api/load-snippet/', { id: id }, function(data) {
    if (data) {
      owner = false;

      langBox.val(data['language']);
      updateLanguage();
      editor.setValue(data['code']);
      editor.getSelection().clearSelection();
    }
  });

  var runCode = function() {
    if (running) return;
    running = true;

    if (!owner) {
      regenerateId();
      reopenEventSource();
    }

    $('#console').text('');
    output = [];
    var params = { id: id, code: editor.getValue(), language: lang };
    var start = new Date();
    $.post('/api/run-snippet/', params, function() {
      running = false;
    });
  }

  langBox.on('change', updateLanguage);
  updateLanguage();

  // Give me cmd+l back!
  editor.commands.removeCommand('gotoline');

  editor.commands.addCommand({
    name: 'runCode',
    bindKey: {
      win: 'Ctrl-Return',
      mac: 'Command-Return'
    },
    exec: runCode
  });

  $('#run').on('click', runCode);
});

