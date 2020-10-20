from mnist import MNIST
import tensorflow as tf
import numpy as np
import h5py
from tensorflow.python.keras.saving.hdf5_format import save_optimizer_weights_to_hdf5_group, load_optimizer_weights_from_hdf5_group


def get_dataset():
    (mnist_images, mnist_labels), _ = \
        tf.keras.datasets.mnist.load_data(path='mnist-%d.npz' % 0)

    dataset = tf.data.Dataset.from_tensor_slices(
        (tf.cast(mnist_images[..., tf.newaxis] / 255.0, tf.float32),
         tf.cast(mnist_labels, tf.int64))
    )
    dataset = dataset.repeat().shuffle(10000).batch(128)
    return dataset


def train():
    model = MNIST()
    optimizer = tf.keras.optimizers.Adam(0.001)

    model.compile(
        optimizer,
        loss=tf.losses.SparseCategoricalCrossentropy()
    )
    dataset = get_dataset()

    model.built = True
    optimizer._create_all_weights(model.trainable_variables)
    model.fit(
        x=dataset.take(5),
        epochs=1
    )

    model.save_weights("my-model-weights.h5", save_format="h5")
    #print("optimizers weights: ", optimizer.get_weights(), len(optimizer.get_weights()))

    with h5py.File("my-optimizer-weights.h5", "w") as h5file:
        opt_group = h5file.create_group("opt")
        save_optimizer_weights_to_hdf5_group(opt_group, optimizer)
        #print(opt_group["optimizer_weights"]["Adam"].keys())
    print("model weights: ", model.get_weights())


def load():
    model = MNIST()

    optimizer = tf.keras.optimizers.Adam(0.000)

    model.compile(
        optimizer,
        loss=tf.losses.SparseCategoricalCrossentropy()
    )

    model.built = True
    model.load_weights("my-model-weights.h5")
    print("My model: ", model.get_weights())
    #print("optimizer: ", d_optimizer.get_weights())

    with h5py.File("my-optimizer-weights.h5", 'r') as h5f:
        weights_group = h5f["opt"]
        optimizer._create_all_weights(model.trainable_variables)
        optimizer_values = load_optimizer_weights_from_hdf5_group(weights_group)
        optimizer.set_weights(optimizer_values)
    # print("optimizer info: ", optimizer.get_weights(), len(optimizer.get_weights()))
    print("optimizer:", optimizer.get_config())
    #optimizer._set_hyper('learning_rate', 0.0)

    dataset = get_dataset()
    model.fit(
        x=dataset.take(5),
        epochs=1
    )
    print("My model: ", model.get_weights())
    print("my metrics: ", model.metrics_names, model.metrics)


if __name__ == "__main__":
    #train()
    load()