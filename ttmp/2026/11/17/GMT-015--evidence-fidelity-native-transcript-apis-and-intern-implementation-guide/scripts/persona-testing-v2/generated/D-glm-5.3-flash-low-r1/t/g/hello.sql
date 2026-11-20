/* sqleton
name: hello
short: Hello test
flags:
  - name: whoami
    type: string
    default: world
    help: Who
sqleton: true
*/
SELECT '{{ .whoami }}' AS greeting
