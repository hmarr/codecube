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

  var runCode = function() {
    var params = { body: editor.getValue(), language: lang };
    $.post('/api/', params, function(data) {
      $('#console').text(data);
    });
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
});

