export default function (el, io) {
  el.addEventListener("click", () => io.send({ on: el.classList.toggle("on") }));
}
