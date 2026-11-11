(() => {
  // pl-mod:0ed31a754f3e:(script)
  function ed31a754f3e_script_default(el) {
    el.setAttribute("data-n", "2");
  }

  // pl-mod:2c9a92056a70:(script)
  function c9a92056a70_script_default(el) {
    el.setAttribute("data-n", "7");
  }

  // pl-mod:2faf461984f8:(script)
  function faf461984f8_script_default(el) {
    el.setAttribute("data-n", "8");
  }

  // pl-mod:5953c78aa402:(script)
  function c78aa402_script_default(el) {
    el.setAttribute("data-n", "9");
  }

  // pl-mod:864ecefaae21:(script)
  function ecefaae21_script_default(el) {
    el.setAttribute("data-n", "1");
  }

  // pl-mod:97eba4753cda:lib/format.ts
  function format(n) {
    return `#${n}`;
  }

  // pl-mod:97eba4753cda:counter.ts
  function eba4753cda_counter_default(el, io) {
    let total = 0;
    io.on((m) => {
      total += m.add;
      io.send({ value: format(total) });
    });
    return () => {
      total = 0;
    };
  }

  // pl-mod:aae5022780a4:(script)
  function aae5022780a4_script_default(el) {
    el.setAttribute("data-n", "3");
  }

  // pl-mod:ae826f548c6b:(script)
  function ae826f548c6b_script_default(el) {
    el.setAttribute("data-n", "0");
  }

  // pl-mod:ba4a7b7b1f1d:(script)
  function ba4a7b7b1f1d_script_default(el) {
    el.setAttribute("data-n", "5");
  }

  // pl-mod:cb564d259b5a:toggle.js
  function cb564d259b5a_toggle_default(el, io) {
    el.addEventListener("click", () => io.send({ on: el.classList.toggle("on") }));
  }

  // pl-mod:ecba03a856a0:(script)
  function ecba03a856a0_script_default(el) {
    el.setAttribute("data-n", "6");
  }

  // pl-mod:ef08fc51f24e:(script)
  function ef08fc51f24e_script_default(el) {
    el.setAttribute("data-n", "4");
  }

  // pl-entry:pl-entry
  define("0ed31a754f3e", ed31a754f3e_script_default);
  define("2c9a92056a70", c9a92056a70_script_default);
  define("2faf461984f8", faf461984f8_script_default);
  define("5953c78aa402", c78aa402_script_default);
  define("864ecefaae21", ecefaae21_script_default);
  define("97eba4753cda", eba4753cda_counter_default);
  define("aae5022780a4", aae5022780a4_script_default);
  define("ae826f548c6b", ae826f548c6b_script_default);
  define("ba4a7b7b1f1d", ba4a7b7b1f1d_script_default);
  define("cb564d259b5a", cb564d259b5a_toggle_default);
  define("ecba03a856a0", ecba03a856a0_script_default);
  define("ef08fc51f24e", ef08fc51f24e_script_default);
})();
