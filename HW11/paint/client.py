import argparse
import ctypes
import socket
from threading import Thread
from tkinter import *

def send_point(last_x, last_y, x, y, sock):
    message = str(last_x) + "," + str(last_y) + "," + str(x) + "," + str(y) + "\n"
    sock.sendall(message.encode())

def mouse_down(event):
    global last_x, last_y
    last_x = event.x
    last_y = event.y
    send_point(last_x, last_y, event.x, event.y, sock)

def mouse_drag(event):
    global last_x, last_y, canvas, sock
    canvas.create_line(last_x, last_y, event.x, event.y, width=2)
    send_point(last_x, last_y, event.x, event.y, sock)
    last_x = event.x
    last_y = event.y

def setup_network(port):
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect(("localhost", port))
    return sock

def receive_points(sock, canvas):
    prevX, prevY = 0, 0
    while True:
        data = sock.recv(1024).decode().strip()
        if not data:
            break
        parts = data.split(",")
        prevX = int(parts[0])
        prevY = int(parts[1])
        x = int(parts[2])
        y = int(parts[3])
        canvas.create_line(prevX, prevY, x, y, width=2)

parser = argparse.ArgumentParser()

def main():
    parser.add_argument('port', nargs='?', const=8081, type=int)
    args = parser.parse_args()

    global canvas, sock
    sock = setup_network(int(args.port))

    root = Tk()
    canvas = Canvas(root, width=500, height=500, bg="white")
    canvas.pack()

    canvas.bind("<Button-1>", mouse_down)
    canvas.bind("<B1-Motion>", mouse_drag)

    receiver_thread = Thread(target=receive_points, args=(sock, canvas))
    receiver_thread.start()

    root.mainloop()

if __name__ == "__main__":
    main()
