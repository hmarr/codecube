var aceMode = function(lang) {
  if (lang === "c") {
    return "c_cpp";
  }
  return lang;
}

$(function() {
  var editor = ace.edit("editor");
  editor.setTheme("ace/theme/monokai");

  var langBox = $('#language');
  var lang = langBox.val();

  var output = [];

  var runCode = function() {
    $('#console').text('');
    output = [];
    var params = { body: editor.getValue(), language: lang };
    var start = new Date();
    $.post('/api/run-code/', params);
  }

  var updateLanguage = function() {
    lang = langBox.val();
    editor.getSession().setMode("ace/mode/" + aceMode(lang));
  }

  langBox.on('change', updateLanguage);
  updateLanguage();

  editor.commands.addCommand({
    name: 'runCode',
    bindKey: {
      win: 'Ctrl-Return',
      mac: 'Command-Return'
    },
    exec: runCode
  });

  $('#run').on('click', runCode);

  var evtSource = new EventSource("/api/event-stream/");
  evtSource.onmessage = function(e) {
    output.push(e.data);
    $('#console').text(output.join('\n'));
  }
});

